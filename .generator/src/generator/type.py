import warnings
from . import formatter
from . import openapi

from .utils import (
    is_primitive,
)


def type_to_go(schema, alternative_name=None, render_nullable=False, render_new=False):
    """Return Go type name for the type."""
    if render_nullable and schema.get("nullable", False):
        prefix = "Nullable"
    else:
        prefix = ""

    # special case for additionalProperties: true
    if schema is True:
        return "interface{}"

    if "enum" not in schema:
        name = formatter.simple_type(
            schema, render_nullable=render_nullable, render_new=render_new
        )
        if name is not None:
            return name

    name = openapi.get_name(schema)
    if name:
        if "enum" in schema:
            return prefix + name
        if (
            not (schema.get("additionalProperties") and not schema.get("properties"))
            and schema.get("type", "object") == "object"
        ):
            return prefix + name

    type_ = schema.get("type")
    if type_ is None:
        if "items" in schema:
            type_ = "array"
        elif "properties" in schema:
            type_ = "object"
        else:
            type_ = "object"
            warnings.warn(
                f"Unknown type for schema: {schema} ({name or alternative_name})"
            )

    if type_ == "array":
        if name and schema.get("x-generate-alias-as-model", False):
            return prefix + name
        if name or alternative_name:
            alternative_name = (name or alternative_name) + "Item"
        name = type_to_go(schema["items"], alternative_name=alternative_name)
        # handle nullable arrays
        if formatter.simple_type(schema["items"]) and schema["items"].get("nullable"):
            name = "*" + name
        return "[]{}".format(name)
    elif type_ == "object":
        if "additionalProperties" in schema:
            return "map[string]{}".format(type_to_go(schema["additionalProperties"]))
        return (
            prefix + alternative_name
            if alternative_name
            and (
                "properties" in schema
                or "oneOf" in schema
                or "anyOf" in schema
                or "allOf" in schema
            )
            else "interface{}"
        )

    raise ValueError(f"Unknown type {type_}")


def get_type_for_parameter(parameter):
    """Return Go type name for the parameter."""
    if "content" in parameter:
        assert "in" not in parameter
        for content in parameter["content"].values():
            return type_to_go(content["schema"])
    return type_to_go(parameter.get("schema"))


def get_type_for_response(response):
    """Return Go type name for the response."""
    if "content" in response:
        for content in response["content"].values():
            if "schema" in content:
                return type_to_go(content["schema"])


def return_type(operation):
    for response in operation.get("responses", {}).values():
        for content in response.get("content", {}).values():
            if "schema" in content:
                return type_to_go(content["schema"]), content["schema"]
        return


def get_schema_from_response(operation):
    response = operation.get("responses", {})
    if "200" in response:
        content = response["200"].get("content", {})
    elif "201" in response:
        content = response["201"].get("content", {})
    elif "default" in response:
        content = response["default"].get("content", {})
    else:
        return {}

    if "application/json" in content:
        schema = content["application/json"].get("schema", {})
    elif "*/*" in content:
        schema = content["*/*"].get("schema", {})
    else:
        return {}

    if schema.get("type") == "array" and "items" in schema:
        return schema["items"]
    
    return schema


def categorize_schema(schema: dict) -> str:
    """
    Categorize the property based on its type.
    """
    if schema and schema.get("type") == "array":
        items = schema.get("items", {})
        if is_primitive(items):
            return "primitive_arr"
        else:
            return "non_primitive_list"
    if is_primitive(schema):
        return "primitive"
    else:
        return "non_primitive_obj"


def sort_schemas_by_type(schemas: dict):
    """
    Sort schemas by primitive and non primitive types since
    we use Blocks in terraform instead of NestedAttributes for
    non primitives.
    """
    # Initialize dictionaries to store different types of parameters
    primitive = {}
    primitive_array = {}
    non_primitive_array = {}
    non_primitive_obj = {}

    # Iterate through the parameters
    for name, schema in schemas.items():
        match categorize_schema(schema["schema"]):
            case "primitive":
                primitive[name] = schema["schema"]
            case "primitive_arr":
                primitive_array[name] = schema["schema"]
            case "non_primitive_list":
                non_primitive_array[name] = schema["schema"]
            case "non_primitive_obj":
                non_primitive_obj[name] = schema["schema"]

    return primitive, primitive_array, non_primitive_array, non_primitive_obj


def tf_sort_params_by_type(parameters):
    """
    Sort parameters by primitive and non primitive types since
    we use Blocks in terraform instead of NestedAttributes for
    non primitives.
    """
    # Initialize dictionaries to store different types of parameters
    primitive = {}
    primitive_array = {}
    non_primitive_array = {}
    non_primitive_obj = {}

    # Iterate through the parameters
    for name, p in parameters.items():
        schema = openapi.parameter_schema(p)
        if openapi.is_json_api(schema):
            # If the schema is a JSON API schema, get the attributes schema
            schema = openapi.json_api_attributes_schema(schema)

        for attr, s in schema.get("properties", {}).items():
            required = attr in schema.get("required", [])
            s["required"] = required

            # categorize the parameter based on its type
            category = categorize_schema(s)
            if category == "primitive":
                primitive[attr] = s
            elif category == "primitive_arr":
                primitive_array[attr] = s
            elif category == "non_primitive_list":
                non_primitive_array[attr] = s
            else:
                non_primitive_obj[attr] = s

    return primitive, primitive_array, non_primitive_array, non_primitive_obj


def tf_sort_properties_by_type(schema):
    """
    Sort schema properties into dictionary by attribute types:
    primitiveAttr, primitiveArrAttr, nonPrimitiveListAttr, nonPrimitiveObjAttr
    """
    primitiveAttr = {}
    primitiveArrAttr = {}
    nonPrimitiveListAttr = {}
    nonPrimitiveObjAttr = {}

    if schema and schema.get("type", "object") == "object":
        for attrName, attrSchema in schema.get("properties", {}).items():
            category = categorize_schema(attrSchema)
            if category == "primitive_arr":
                primitiveArrAttr[attrName] = attrSchema
            elif category == "primitive":
                primitiveAttr[attrName] = attrSchema
            elif category == "non_primitive_list":
                nonPrimitiveListAttr[attrName] = attrSchema
            elif category == "non_primitive_obj":
                nonPrimitiveObjAttr[attrName] = attrSchema

    return primitiveAttr, primitiveArrAttr, nonPrimitiveListAttr, nonPrimitiveObjAttr
