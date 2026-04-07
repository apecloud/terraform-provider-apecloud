import os
import pathlib
import click
import subprocess

from jinja2 import Template

from . import setup
from . import openapi


@click.command()
@click.argument(
    "spec_path",
    type=click.Path(
        exists=True, file_okay=True, dir_okay=False, path_type=pathlib.Path
    ),
)
@click.argument(
    "config_path",
    type=click.Path(
        exists=True, file_okay=True, dir_okay=False, path_type=pathlib.Path
    ),
)
@click.option("--go-fmt/--no-go-fmt", default=True)
def cli(spec_path, config_path, go_fmt):
    """
    Generate terraform model structs from OpenAPI specification.
    """
    env = setup.load_environment(version=spec_path.parent.name)

    templates = setup.load_templates(env=env)

    spec = setup.load(spec_path)
    config = setup.load(config_path)

    resources_to_generate = openapi.get_resources(spec, config)
    for name, resource in resources_to_generate.items():
        generate_resource(
            name=name,
            resource=resource,
            templates=templates,
            go_fmt=go_fmt,
        )

    datasources_to_generate = openapi.get_data_sources(spec, config)
    for name, datasource in datasources_to_generate.items():
        generate_datasource(
            name=name,
            datasource=datasource,
            templates=templates,
            go_fmt=go_fmt,
        )

def write_and_fmt(filename: pathlib.Path, content: str, go_fmt: bool):
    if not filename.parent.exists():
        os.makedirs(filename.parent)
    with filename.open("w") as fp:
        fp.write(content)
    if go_fmt:
        subprocess.call(["go", "fmt", filename])


def generate_resource(
    name: str, resource: dict, templates: dict[str, Template], go_fmt: bool
) -> None:
    types_output = pathlib.Path("../../internal/types/")
    filename = types_output / f"resource_{name}_model_gen.go"
    write_and_fmt(filename, templates["resource_model"].render(name=name, operations=resource), go_fmt)

    resource_output = pathlib.Path("../../internal/resource/")
    filename = resource_output / f"resource_{name}_schema_gen.go"
    write_and_fmt(filename, templates["resource_schema"].render(name=name, operations=resource), go_fmt)


def generate_datasource(
    name: str, datasource: dict, templates: dict[str, Template], go_fmt: bool
) -> None:
    types_output = pathlib.Path("../../internal/types/")
    filename = types_output / f"datasource_{name}_model_gen.go"
    write_and_fmt(filename, templates["datasource_model"].render(name=name, operations=datasource), go_fmt)

    datasource_output = pathlib.Path("../../internal/datasource/")
    filename = datasource_output / f"datasource_{name}_schema_gen.go"
    write_and_fmt(filename, templates["datasource_schema"].render(name=name, operations=datasource), go_fmt)

if __name__ == "__main__":
    cli()

