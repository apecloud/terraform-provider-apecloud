package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/spf13/cast"

	"github.com/apecloud/kb-cloud-client-go/api/kbcloud"
	"github.com/apecloud/kb-cloud-client-go/api/kbcloud/admin"

	"github.com/apecloud/terraform-provider-kbcloud/internal/client"
	mytypes "github.com/apecloud/terraform-provider-kbcloud/internal/types"
	"github.com/apecloud/terraform-provider-kbcloud/internal/utils"
	"github.com/apecloud/terraform-provider-kbcloud/internal/utils/pointer"
)

type ClusterResource struct {
	client *client.Client
}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

func (r *ClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ClustersResourceSchema()
}

func (r *ClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data mytypes.ClustersResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := clusterResourceToClusterCreate(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var (
		apiData      map[string]interface{}
		clusterBytes []byte
		err          error
	)
	if r.client.IsAdminClient() {
		cluster, apiResp, err := admin.NewClusterApi(r.client.AdminClient()).CreateCluster(r.client.AdminCtx(), data.OrgName.ValueString(), pointer.ValueOf(body))
		if err != nil {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error:%s %s", err.Error(), errDetail))
			return
		}

		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error:%s", errDetail))
			return
		}
		data.ID = types.StringValue(pointer.ValueOf(cluster.Id))
		clusterBytes, err = json.Marshal(cluster)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster, got error:%s", err.Error()))
			return
		}
		if err := json.Unmarshal(clusterBytes, &apiData); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster, got error:%s", err.Error()))
			return
		}
		data.ID = types.StringValue(pointer.ValueOf(cluster.Id))
		diags = data.RefreshFromAPI(ctx, apiData)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	} else {
		apiBody := kbcloud.ClusterCreate{}
		clusterBytes, err = json.Marshal(body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster body, got error:%s ", err.Error()))
			return
		}
		if err := json.Unmarshal(clusterBytes, &apiBody); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster body, got error:%s", err.Error()))
			return
		}

		cluster, apiResp, err := kbcloud.NewClusterApi(r.client.Client()).CreateCluster(r.client.Ctx(), data.OrgName.ValueString(), apiBody)
		if err != nil {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error:%s", errDetail))
			return
		}

		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error:%s", errDetail))
			return
		}
		data.ID = types.StringValue(pointer.ValueOf(cluster.Id))
		if err := json.Unmarshal(clusterBytes, &apiData); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster, got error:%s", err.Error()))
			return
		}
		diags = data.RefreshFromAPI(ctx, apiData)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Call API and set state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data mytypes.ClustersResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.OrgName.IsNull() || data.Name.IsNull() || data.OrgName.IsUnknown() || data.Name.IsUnknown() {
		id := data.ID.ValueString()
		resp.Diagnostics.AddWarning(id, fmt.Sprintf("cluster resource (%s) is unknown, clear it from state", id))
		resp.State.RemoveResource(ctx)
		return
	}

	// update cluster state by get api
	diags = r.updateClusterState(ctx, &data)
	if diags.HasError() {
		if diags.Contains(utils.NewNotFoundErrorDiagnostic(data.Name.ValueString(), data.OrgName.ValueString())) {
			resp.State.RemoveResource(ctx)
		}
		resp.Diagnostics.Append(diag.NewWarningDiagnostic("cluster not found", fmt.Sprintf("cluster %s in org %s not found", data.Name.ValueString(), data.OrgName.ValueString())))
		return
	}
	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// updateClusterState updates cluster state by get api
func (r *ClusterResource) updateClusterState(ctx context.Context, data *mytypes.ClustersResourceModel) diag.Diagnostics {
	var (
		apiData      map[string]interface{}
		diags        diag.Diagnostics
		clusterBytes []byte
		err          error
		apiResp      *http.Response
		cluster      interface{}
	)

	if r.client.IsAdminClient() {
		cluster, apiResp, err = admin.NewClusterApi(r.client.AdminClient()).GetCluster(r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString())
	} else {
		cluster, apiResp, err = kbcloud.NewClusterApi(r.client.Client()).GetCluster(r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString())
	}

	if apiResp != nil && apiResp.StatusCode == 404 {
		diags = append(diags, utils.NewNotFoundErrorDiagnostic(data.Name.ValueString(), data.OrgName.ValueString()))
		return diags
	}

	if !utils.IsHTTPSuccess(apiResp) || err != nil {
		errDetail := utils.GetRespErrorDetail(apiResp)
		diags = append(diags, diag.NewErrorDiagnostic(
			"read cluster error",
			fmt.Sprintf("read cluster %s in org %s error: %v %s", data.Name.ValueString(), data.OrgName.ValueString(), err, errDetail),
		))
		return diags
	}
	clusterBytes, err = json.Marshal(cluster)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster, got error:%s", err.Error()))
		return diags
	}
	if err := json.Unmarshal(clusterBytes, &apiData); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster, got error:%s", err.Error()))
		return diags
	}

	data.ID = types.StringValue(apiData["id"].(string))
	diags.Append(data.RefreshFromAPI(ctx, apiData)...)
	return diags
}

// mergeClusterState merges the latest remote state with the local state
// the merge policy is to:
// 1. if local state field is nil or unknown, use the remote state field
// 2. if local state field is not nil or unknown, keep the local state field
func (r *ClusterResource) mergeClusterState(ctx context.Context, data *mytypes.ClustersResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get latest remote state
	var (
		clusterBytes []byte
		err          error
		apiResp      *http.Response
		cluster      interface{}
		apiData      map[string]interface{}
	)

	if r.client.IsAdminClient() {
		cluster, apiResp, err = admin.NewClusterApi(r.client.AdminClient()).GetCluster(r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString())
	} else {
		cluster, apiResp, err = kbcloud.NewClusterApi(r.client.Client()).GetCluster(r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString())
	}

	if apiResp != nil && apiResp.StatusCode == 404 {
		diags = append(diags, utils.NewNotFoundErrorDiagnostic(data.Name.ValueString(), data.OrgName.ValueString()))
		return diags
	}

	if !utils.IsHTTPSuccess(apiResp) || err != nil {
		errDetail := utils.GetRespErrorDetail(apiResp)
		diags = append(diags, diag.NewErrorDiagnostic(
			"read cluster error",
			fmt.Sprintf("read cluster %s in org %s error: %v %s", data.Name.ValueString(), data.OrgName.ValueString(), err, errDetail),
		))
		return diags
	}

	clusterBytes, err = json.Marshal(cluster)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster, got error:%s", err.Error()))
		return diags
	}
	if err := json.Unmarshal(clusterBytes, &apiData); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster, got error:%s", err.Error()))
		return diags
	}

	remoteData := mytypes.ClustersResourceModel{}
	remoteData.RefreshFromAPI(ctx, apiData)

	// Merge strategy:
	// 1. if local state field is nil or unknown, use the remote state field
	// 2. if local state field is not nil or unknown, keep the local state field
	//
	// We call the auto-generated Merge function on the top level data model.
	data.Merge(&remoteData)

	return diags
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data mytypes.ClustersResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defer func() {
		refreshDiags := r.mergeClusterState(ctx, &data)
		diags.Append(refreshDiags...)

		// It is actually recommended in Terraform to save the state even if there are errors
		// during an update, so that Terraform knows exactly what successfully changed.
		// If mergeClusterState succeeded, data now contains the real world state.
		if !refreshDiags.HasError() {
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}

		// Finally, make sure all collected diags are appended to the response
		resp.Diagnostics.Append(diags...)
	}()

	body, updateDiags := clusterResourceToClusterUpdate(ctx, &data)
	diags.Append(updateDiags...)
	if diags.HasError() {
		return
	}

	var (
		clusterBytes  []byte
		err           error
		oldComponents []admin.ComponentItem
	)

	// 1. Update cluster by UpdateAPI
	if r.client.IsAdminClient() {
		_, apiResp, err := admin.NewClusterApi(r.client.AdminClient()).PatchCluster(r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString(), pointer.ValueOf(body))
		if err != nil {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error: %s", errDetail))
			return
		}
	} else {
		apiBody := kbcloud.ClusterUpdate{}
		clusterBytes, err = json.Marshal(body)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster body, got error: %s", err.Error()))
			return
		}
		if err := json.Unmarshal(clusterBytes, &apiBody); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster body, got error:%s", err.Error()))
			return
		}

		_, apiResp, err := kbcloud.NewClusterApi(r.client.Client()).PatchCluster(r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString(), apiBody)
		if err != nil {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error %s", errDetail))
			return
		}
	}

	// 2. Update cluster components info By Ops API
	if r.client.IsAdminClient() {
		cluster, apiResp, err := admin.NewClusterApi(r.client.AdminClient()).GetCluster(r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString())
		if err != nil {
			if apiResp != nil && apiResp.StatusCode == 404 {
				resp.State.RemoveResource(ctx)
				return
			}
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		oldComponents = cluster.Components
		data.ID = types.StringValue(pointer.ValueOf(cluster.Id))
	} else {
		cluster, apiResp, err := kbcloud.NewClusterApi(r.client.Client()).GetCluster(r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString())
		if err != nil {
			if apiResp != nil && apiResp.StatusCode == 404 {
				resp.State.RemoveResource(ctx)
				return
			}
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		clusterBytes, err = json.Marshal(cluster)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal cluster, got error:%s", err.Error()))
			return
		}
		var oldCluster admin.Cluster
		err = json.Unmarshal(clusterBytes, &oldCluster)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal cluster, got error:%s", err.Error()))
			return
		}
		oldComponents = oldCluster.Components
		data.ID = types.StringValue(pointer.ValueOf(cluster.Id))
	}

	var components []*mytypes.ComponentsResourceModel
	if !data.Components.IsNull() && !data.Components.IsUnknown() {
		diags.Append(data.Components.ElementsAs(ctx, &components, false)...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	for _, cmp := range components {
		diags.Append(r.doVscale(oldComponents, &data, cmp)...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		diags.Append(r.doHscale(oldComponents, &data, cmp)...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		diags.Append(r.doVolumeExpand(oldComponents, &data, cmp)...)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	resp.Diagnostics.Append(diags...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data mytypes.ClustersResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if r.client.IsAdminClient() {
		_, apiResp, err := admin.NewClusterApi(r.client.AdminClient()).DeleteCluster(r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString())
		if err != nil {
			if apiResp != nil && apiResp.StatusCode == 404 {
				return
			}
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error %s", errDetail))
			return
		}
	} else {
		_, apiResp, err := kbcloud.NewClusterApi(r.client.Client()).DeleteCluster(r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString())
		if err != nil {
			if apiResp != nil && apiResp.StatusCode == 404 {
				return
			}
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error:%s %s", err.Error(), errDetail))
			return
		}
		if !utils.IsHTTPSuccess(apiResp) {
			errDetail := utils.GetRespErrorDetail(apiResp)
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error: %s", errDetail))
			return
		}
	}
}

func clusterResourceToClusterCreate(ctx context.Context, data *mytypes.ClustersResourceModel) (*admin.ClusterCreate, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	body := admin.ClusterCreate{
		Name:            data.Name.ValueString(),
		EnvironmentName: data.EnvironmentName.ValueString(),
		OrgName:         data.OrgName.ValueStringPointer(),
		Project:         data.Project.ValueStringPointer(),
		Engine:          data.Engine.ValueString(),
		Version:         data.Version.ValueStringPointer(),
		Mode:            data.Mode.ValueStringPointer(),
		SingleZone:      data.SingleZone.ValueBoolPointer(),
		DisplayName:     data.DisplayName.ValueStringPointer(),
		Static:          data.Static.ValueBoolPointer(),
	}
	if !data.NetworkMode.IsNull() && !data.NetworkMode.IsUnknown() {
		body.NetworkMode = admin.NetworkMode(data.NetworkMode.ValueString()).Ptr()
	}

	if data.ClusterType.ValueString() == "Normal" {
		body.ClusterType.Set(admin.ClusterTypeNormal.Ptr())
	} else if data.ClusterType.ValueString() == "DisasterRecovery" {
		body.ClusterType.Set(admin.ClusterTypeDisasterRecovery.Ptr())
	}

	if data.TerminationPolicy.ValueString() == "DoNotTerminate" {
		body.TerminationPolicy = admin.ClusterTerminationPolicyDoNotTerminate.Ptr()
	} else {
		body.TerminationPolicy = admin.ClusterTerminationPolicyDelete.Ptr()
	}

	if !data.Extra.IsNull() && !data.Extra.IsUnknown() {
		// In Terraform Plugin Framework, types.Dynamic holds arbitrary underlying types
		// (e.g. types.Map, types.Object). However, basetypes.ObjectValue doesn't
		// implement json.Marshaler natively for its internal attributes.
		// We need to recursively extract the values into pure Go types.
		extraMap := utils.ExtractAttrValue(ctx, data.Extra)
		if extraMap != nil {
			if m, ok := extraMap.(map[string]interface{}); ok && len(m) > 0 {
				body.Extra = m
			}
		}
	}

	if !data.NodeGroup.IsNull() && !data.NodeGroup.IsUnknown() {
		body.NodeGroup.Set(data.NodeGroup.ValueStringPointer())
	}

	if !data.AvailabilityZones.IsNull() && !data.AvailabilityZones.IsUnknown() {
		var availabilityZones []string
		diags = data.AvailabilityZones.ElementsAs(ctx, &availabilityZones, true)
		if diags.HasError() {
			return nil, diags
		}
		body.AvailabilityZones = availabilityZones
	}

	var paramTpls []*mytypes.ParamTplsResourceModel
	if !data.ParamTpls.IsNull() && !data.ParamTpls.IsUnknown() {
		diags.Append(data.ParamTpls.ElementsAs(ctx, &paramTpls, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	var apiParamTpls []admin.ParamTplsItem
	if !data.ParamTpls.IsNull() && !data.ParamTpls.IsUnknown() {
		for _, paramTpl := range paramTpls {
			apiParamTpls = append(apiParamTpls, admin.ParamTplsItem{
				Component:         paramTpl.Component.ValueStringPointer(),
				ParamTplName:      paramTpl.ParamTplName.ValueStringPointer(),
				ParamTplPartition: admin.ParameterTemplatePartition(paramTpl.ParamTplPartition.ValueString()).Ptr(),
			})
		}
	} else {
		// explicitly use an empty slice rather than nil so JSON marshal generates `[]`
		apiParamTpls = make([]admin.ParamTplsItem, 0)
	}

	body.ParamTpls = apiParamTpls

	var components []*mytypes.ComponentsResourceModel
	if !data.Components.IsNull() && !data.Components.IsUnknown() {
		diags.Append(data.Components.ElementsAs(ctx, &components, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	var apiComponents []admin.ComponentItemCreate
	for _, component := range components {
		var volumes []*mytypes.VolumesResourceModel
		if !component.Volumes.IsNull() && !component.Volumes.IsUnknown() {
			diags.Append(component.Volumes.ElementsAs(ctx, &volumes, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}
		var apiVolumes []admin.ComponentVolumeItem
		for _, volume := range volumes {
			v := admin.ComponentVolumeItem{
				Name:    volume.Name.ValueStringPointer(),
				Storage: volume.Storage.ValueFloat64Pointer(),
			}

			if !volume.IoLimits.IsNull() && !volume.IoLimits.IsUnknown() {
				var ioLimits mytypes.IoLimitsResourceModel
				diags.Append(volume.IoLimits.As(ctx, &ioLimits, basetypes.ObjectAsOptions{})...)
				if !diags.HasError() {
					v.IoLimits = &admin.IoQuantity{}
					if !ioLimits.ReadIops.IsNull() && !ioLimits.ReadIops.IsUnknown() {
						v.IoLimits.ReadIops.Set(ioLimits.ReadIops.ValueInt64Pointer())
					}
					if !ioLimits.WriteIops.IsNull() && !ioLimits.WriteIops.IsUnknown() {
						v.IoLimits.WriteIops.Set(ioLimits.WriteIops.ValueInt64Pointer())
					}
					if !ioLimits.ReadBps.IsNull() && !ioLimits.ReadBps.IsUnknown() {
						v.IoLimits.ReadBps.Set(ioLimits.ReadBps.ValueInt64Pointer())
					}
					if !ioLimits.WriteBps.IsNull() && !ioLimits.WriteBps.IsUnknown() {
						v.IoLimits.WriteBps.Set(ioLimits.WriteBps.ValueInt64Pointer())
					}
				}
			}

			if !volume.IoReserves.IsNull() && !volume.IoReserves.IsUnknown() {
				var ioReserves mytypes.IoReservesResourceModel
				diags.Append(volume.IoReserves.As(ctx, &ioReserves, basetypes.ObjectAsOptions{})...)
				if !diags.HasError() {
					v.IoReserves = &admin.IoQuantity{}
					if !ioReserves.ReadIops.IsNull() && !ioReserves.ReadIops.IsUnknown() {
						v.IoReserves.ReadIops.Set(ioReserves.ReadIops.ValueInt64Pointer())
					}
					if !ioReserves.WriteIops.IsNull() && !ioReserves.WriteIops.IsUnknown() {
						v.IoReserves.WriteIops.Set(ioReserves.WriteIops.ValueInt64Pointer())
					}
					if !ioReserves.ReadBps.IsNull() && !ioReserves.ReadBps.IsUnknown() {
						v.IoReserves.ReadBps.Set(ioReserves.ReadBps.ValueInt64Pointer())
					}
					if !ioReserves.WriteBps.IsNull() && !ioReserves.WriteBps.IsUnknown() {
						v.IoReserves.WriteBps.Set(ioReserves.WriteBps.ValueInt64Pointer())
					}
				}
			}
			apiVolumes = append(apiVolumes, v)
		}

		cmp := admin.ComponentItemCreate{
			Volumes:   apiVolumes,
			Component: component.Component.ValueString(),
		}
		if !component.ClassCode.IsNull() && !component.ClassCode.IsUnknown() {
			cmp.ClassCode = component.ClassCode.ValueStringPointer()
		}
		if !component.SystemAccountSecretName.IsNull() && !component.SystemAccountSecretName.IsUnknown() {
			cmp.SystemAccountSecretName = component.SystemAccountSecretName.ValueStringPointer()
		}

		if !component.Cpu.IsNull() && !component.Cpu.IsUnknown() {
			cmp.Cpu = component.Cpu.ValueFloat64Pointer()
		}
		if !component.Memory.IsNull() && !component.Memory.IsUnknown() {
			cmp.Memory = component.Memory.ValueFloat64Pointer()
		}
		if !component.StorageClass.IsNull() && !component.StorageClass.IsUnknown() {
			cmp.StorageClass = component.StorageClass.ValueStringPointer()
		}
		if !component.CompNum.IsNull() && !component.CompNum.IsUnknown() {
			cmp.CompNum = pointer.Int32(int32(component.CompNum.ValueInt64()))
		}
		if !component.Replicas.IsNull() && !component.Replicas.IsUnknown() {
			cmp.Replicas = pointer.Int32(int32(component.Replicas.ValueInt64()))
		}
		if !component.ClassCode.IsNull() && !component.ClassCode.IsUnknown() {
			cmp.ClassCode = component.ClassCode.ValueStringPointer()
		}
		apiComponents = append(apiComponents, cmp)
	}
	body.Components = apiComponents

	var initOptions []*mytypes.InitOptionsResourceModel
	if !data.InitOptions.IsNull() && !data.InitOptions.IsUnknown() {
		diags.Append(data.InitOptions.ElementsAs(ctx, &initOptions, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	var apiInitOptions []admin.InitOptionItem
	for _, initOption := range initOptions {
		initParams := make(map[string]string)
		diags = initOption.InitParams.ElementsAs(ctx, &initParams, true)
		if diags.HasError() {
			return nil, diags
		}
		apiInitOptions = append(apiInitOptions, admin.InitOptionItem{
			Component:  initOption.Component.ValueStringPointer(),
			InitParams: initParams,
			SpecName:   initOption.SpecName.ValueStringPointer(),
		})
	}
	body.InitOptions = apiInitOptions

	var serviceRefs []*mytypes.ServiceRefsResourceModel
	if !data.ServiceRefs.IsNull() && !data.ServiceRefs.IsUnknown() {
		diags.Append(data.ServiceRefs.ElementsAs(ctx, &serviceRefs, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	var apiServiceRefs []admin.ServiceRef
	for _, serviceRef := range serviceRefs {
		svcRef := admin.ServiceRef{
			Name:    serviceRef.Name.ValueString(),
			Cluster: serviceRef.Cluster.ValueStringPointer(),
		}
		if !serviceRef.ServiceDescriptor.IsNull() && !serviceRef.ServiceDescriptor.IsUnknown() {
			var sd mytypes.ServiceDescriptorResourceModel
			diags.Append(serviceRef.ServiceDescriptor.As(ctx, &sd, basetypes.ObjectAsOptions{})...)
			if !diags.HasError() {
				svcRef.ServiceDescriptor = &admin.ServiceDescriptor{
					Host:     sd.Host.ValueStringPointer(),
					Port:     sd.Port.ValueStringPointer(),
					Endpoint: sd.Endpoint.ValueStringPointer(),
					Username: sd.Username.ValueStringPointer(),
					Password: sd.Password.ValueStringPointer(),
				}
			}
		}
		apiServiceRefs = append(apiServiceRefs, svcRef)
	}
	body.ServiceRefs = apiServiceRefs

	if !data.License.IsNull() && !data.License.IsUnknown() {
		var license mytypes.LicenseResourceModel
		diags.Append(data.License.As(ctx, &license, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			lic := &admin.ClusterLicense{
				Id:   license.Id.ValueStringPointer(),
				Name: license.Name.ValueStringPointer(),
			}
			if !license.ExpiredAt.IsNull() {
				expiredAt, err := time.Parse(time.RFC3339, license.ExpiredAt.ValueString())
				if err != nil {
					diags.AddError("Client Error", fmt.Sprintf("Unable to parse expiredAt, got error: %s", err))
					return nil, diags
				}
				lic.ExpiredAt = pointer.To(expiredAt)
			}
			if !license.Key.IsNull() {
				filePath := license.Key.ValueString()
				file, err := os.Open(filePath)
				if err != nil {
					diags.AddError(
						"open license key file failed when create cluster",
						fmt.Sprintf("Unable to open license key file at %s: %s", filePath, err.Error()),
					)
					return nil, diags
				}
				defer file.Close()
				var reader io.Reader = file
				lic.Key = &reader
			}
			body.License = lic
		}
	}

	if !data.Backup.IsNull() && !data.Backup.IsUnknown() {
		var backupModel mytypes.BackupResourceModel
		diags.Append(data.Backup.As(ctx, &backupModel, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			backup := &admin.ClusterBackup{
				PitrEnabled: backupModel.PitrEnabled.ValueBoolPointer(),
				// continuous backup method for pitr
				ContinuousBackupMethod: backupModel.ContinuousBackupMethod.ValueStringPointer(),
				// autoBackup or not
				AutoBackup: backupModel.AutoBackup.ValueBoolPointer(),
				// name of the backup method
				AutoBackupMethod: backupModel.AutoBackupMethod.ValueStringPointer(),
				// backupRepoName is the name of backupRepo and it is used to store the backup data
				BackupRepo: backupModel.BackupRepo.ValueStringPointer(),
				// cronExpression specifies the cron expression
				CronExpression: backupModel.CronExpression.ValueStringPointer(),
				// retentionPeriod specifies the retention period
				RetentionPeriod: backupModel.RetentionPeriod.ValueStringPointer(),
				// specify whether to enable incremental backup
				IncrementalBackupEnabled: backupModel.IncrementalBackupEnabled.ValueBoolPointer(),
				// the crop expression for incremental backup schedule
				IncrementalCronExpression: backupModel.IncrementalCronExpression.ValueStringPointer(),
			}
			if !backupModel.RetentionPolicy.IsNull() && !backupModel.RetentionPolicy.IsUnknown() {
				backup.RetentionPolicy = admin.BackupRetentionPolicy(backupModel.RetentionPolicy.ValueString()).Ptr()
			}
			if !backupModel.SnapshotVolumes.IsNull() && !backupModel.SnapshotVolumes.IsUnknown() {
				backup.SnapshotVolumes.Set(backupModel.SnapshotVolumes.ValueBoolPointer())
			}
			body.Backup = backup
		}
	}

	if !data.ObjectStorageConfig.IsNull() && !data.ObjectStorageConfig.IsUnknown() {
		var objStore mytypes.ObjectStorageConfigResourceModel
		diags.Append(data.ObjectStorageConfig.As(ctx, &objStore, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			objectStorageConfig := &admin.ClusterObjectStorageConfig{
				Bucket: objStore.Bucket.ValueString(),
				// whether the object storage is using path-style. If false, virtual host style will be used.
				UsePathStyle: objStore.UsePathStyle.ValueBoolPointer(),
				// region to use. If using a s3-compatible service that does not require a region (like minio), leave it blank.
				Region: objStore.Region.ValueStringPointer(),
			}
			if !objStore.ServiceRef.IsNull() && !objStore.ServiceRef.IsUnknown() {
				var svcRefModel mytypes.ServiceRefResourceModel
				diags.Append(objStore.ServiceRef.As(ctx, &svcRefModel, basetypes.ObjectAsOptions{})...)
				if !diags.HasError() {
					objSvcRef := admin.ServiceRef{
						Name:    svcRefModel.Name.ValueString(),
						Cluster: svcRefModel.Cluster.ValueStringPointer(),
					}
					if !svcRefModel.ServiceDescriptor.IsNull() && !svcRefModel.ServiceDescriptor.IsUnknown() {
						var sd mytypes.ServiceDescriptorResourceModel
						diags.Append(svcRefModel.ServiceDescriptor.As(ctx, &sd, basetypes.ObjectAsOptions{})...)
						if !diags.HasError() {
							objSvcRef.ServiceDescriptor = &admin.ServiceDescriptor{
								Host:     sd.Host.ValueStringPointer(),
								Port:     sd.Port.ValueStringPointer(),
								Endpoint: sd.Endpoint.ValueStringPointer(),
								Username: sd.Username.ValueStringPointer(),
								Password: sd.Password.ValueStringPointer(),
							}
						}
					}
					objectStorageConfig.ServiceRef = objSvcRef
				}
			}
			body.ObjectStorageConfig = objectStorageConfig
		}
	}

	if !data.MaintainceWindow.IsNull() && !data.MaintainceWindow.IsUnknown() {
		var mw mytypes.MaintainceWindowResourceModel
		diags.Append(data.MaintainceWindow.As(ctx, &mw, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			maintainceWindow := &admin.ClusterMaintainceWindow{
				StartHour: pointer.To(int32(mw.StartHour.ValueInt64())),
				EndHour:   pointer.To(int32(mw.EndHour.ValueInt64())),
				Weekdays:  mw.Weekdays.ValueStringPointer(),
			}
			body.MaintainceWindow = maintainceWindow
		}
	}

	return &body, diags
}

// clusterResourceToClusterUpdate converts cluster update body from resource model
func clusterResourceToClusterUpdate(ctx context.Context, data *mytypes.ClustersResourceModel) (*admin.ClusterUpdate, diag.Diagnostics) {
	var (
		update admin.ClusterUpdate
		diags  diag.Diagnostics
	)

	// Since some fields might be un-configured (null) or set explicitly in the new plan,
	// we need to set them properly.
	if !data.TerminationPolicy.IsNull() && !data.TerminationPolicy.IsUnknown() {
		update.TerminationPolicy = admin.ClusterTerminationPolicy(data.TerminationPolicy.ValueString()).Ptr()
	} else {
		// default is DoNotTerminate
		update.TerminationPolicy = admin.ClusterTerminationPolicyDoNotTerminate.Ptr()
	}

	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		update.DisplayName.Set(data.DisplayName.ValueStringPointer())
	} else {
		// default is cluster name
		update.DisplayName.Set(data.Name.ValueStringPointer())
	}

	if !data.MaintainceWindow.IsNull() && !data.MaintainceWindow.IsUnknown() {
		var mw mytypes.MaintainceWindowResourceModel
		diags.Append(data.MaintainceWindow.As(ctx, &mw, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			update.MaintainceWindow = &admin.ClusterMaintainceWindow{
				StartHour: pointer.To(int32(mw.StartHour.ValueInt64())),
				EndHour:   pointer.To(int32(mw.EndHour.ValueInt64())),
				Weekdays:  mw.Weekdays.ValueStringPointer(),
			}
		}
	} else {
		// default is every day 02:00-06:00 in UTC
		update.MaintainceWindow = &admin.ClusterMaintainceWindow{
			StartHour: pointer.To(int32(2)),
			EndHour:   pointer.To(int32(6)),
			Weekdays:  pointer.To("1,2,3,4,5,6,7"),
		}
	}

	return &update, diags
}

func (r *ClusterResource) doVscale(oldComponents []admin.ComponentItem, data *mytypes.ClustersResourceModel, update *mytypes.ComponentsResourceModel) diag.Diagnostics {
	var (
		diags diag.Diagnostics
	)

	// no need to update
	if !slices.ContainsFunc(oldComponents, func(c admin.ComponentItem) bool {
		return pointer.ValueOf(c.Component) == update.Component.ValueString() &&
			(pointer.ValueOf(c.Cpu) != update.Cpu.ValueFloat64() && update.Cpu.ValueFloat64() != 0 ||
				pointer.ValueOf(c.Memory) != update.Memory.ValueFloat64() && update.Memory.ValueFloat64() != 0 ||
				pointer.ValueOf(c.ClassCode) != update.ClassCode.ValueString() && update.ClassCode.ValueString() != "")
	}) {
		return diags
	}

	var apiResp *http.Response
	var err error

	if r.client.IsAdminClient() {
		_, apiResp, err = admin.NewOpsrequestApi(r.client.AdminClient()).VerticalScaleCluster(
			r.client.AdminCtx(),
			data.OrgName.ValueString(),
			data.Name.ValueString(),
			admin.OpsVScale{
				Component: update.Component.ValueString(),
				Cpu:       pointer.To(cast.ToString(update.Cpu.ValueFloat64())),
				Memory:    pointer.To(cast.ToString(update.Memory.ValueFloat64())),
				ClassCode: update.ClassCode.ValueStringPointer(),
			},
		)
	} else {
		_, apiResp, err = kbcloud.NewOpsrequestApi(r.client.Client()).VerticalScaleCluster(
			r.client.Ctx(),
			data.OrgName.ValueString(),
			data.Name.ValueString(),
			kbcloud.OpsVScale{
				Component: update.Component.ValueString(),
				Cpu:       pointer.To(cast.ToString(update.Cpu.ValueFloat64())),
				Memory:    pointer.To(cast.ToString(update.Memory.ValueFloat64())),
				ClassCode: update.ClassCode.ValueStringPointer(),
			},
		)
	}

	if err != nil {
		errDetail := utils.GetRespErrorDetail(apiResp)
		diags.AddError("update cluster failed when vertical scale", errDetail)
	}

	return diags
}

func (r *ClusterResource) doHscale(oldComponents []admin.ComponentItem, data *mytypes.ClustersResourceModel, update *mytypes.ComponentsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// valuePrecheck
	shardPrecheck := func(newVal types.Int64) bool {
		return slices.ContainsFunc(oldComponents, func(c admin.ComponentItem) bool {
			return pointer.ValueOf(c.Component) == update.Component.ValueString() &&
				(pointer.ValueOf(c.CompNum) != int32(newVal.ValueInt64()) && newVal.ValueInt64() != 0)
		})
	}

	replicasPrecheck := func(newVal types.Int64) bool {
		return slices.ContainsFunc(oldComponents, func(c admin.ComponentItem) bool {
			return pointer.ValueOf(c.Component) == update.Component.ValueString() &&
				(pointer.ValueOf(c.Replicas) != int32(newVal.ValueInt64()) && newVal.ValueInt64() != 0)
		})
	}

	shardChanged := !update.CompNum.IsNull() && !update.CompNum.IsUnknown() && shardPrecheck(update.CompNum)
	replicasChanged := !update.Replicas.IsNull() && !update.Replicas.IsUnknown() && replicasPrecheck(update.Replicas)

	if !shardChanged && !replicasChanged {
		return diags
	}

	var targetBackup string
	var apiResp *http.Response
	var err error

	// Helper to get target backup
	getTargetBackup := func(backups []admin.Backup) string {
		if len(backups) == 0 {
			return ""
		}
		sort.Slice(backups, func(i, j int) bool {
			iCompleted := string(backups[i].Status) == string(admin.BackupStatusCompleted)
			jCompleted := string(backups[j].Status) == string(admin.BackupStatusCompleted)

			if iCompleted != jCompleted {
				return iCompleted
			}
			timeI := pointer.ValueOf(backups[i].CompletionTimestamp.Get())
			timeJ := pointer.ValueOf(backups[j].CompletionTimestamp.Get())
			return timeI.After(timeJ)
		})

		if string(backups[0].Status) == string(admin.BackupStatusCompleted) {
			return backups[0].Name
		}
		return ""
	}

	// Helper to fetch backups depending on client
	var listResp *http.Response
	var listErr error
	var adminBackups []admin.Backup

	if r.client.IsAdminClient() {
		backups, resp, err := admin.NewBackupApi(r.client.AdminClient()).ListBackups(
			r.client.AdminCtx(),
			admin.ListBackupsOptionalParameters{ClusterId: pointer.To(data.ID.ValueString())},
		)
		listResp, listErr = resp, err
		if err == nil {
			adminBackups = backups.Items
		}
	} else {
		backups, resp, err := kbcloud.NewBackupApi(r.client.Client()).ListBackups(
			r.client.Ctx(),
			data.OrgName.ValueString(),
			kbcloud.ListBackupsOptionalParameters{ClusterId: pointer.To(data.ID.ValueString())},
		)
		listResp, listErr = resp, err
		if err == nil {
			b, _ := json.Marshal(backups.Items)
			_ = json.Unmarshal(b, &adminBackups)
		}
	}

	if listErr != nil || !utils.IsHTTPSuccess(listResp) {
		errDetail := utils.GetRespErrorDetail(listResp)
		diags.AddWarning("List backups By Cluster failed during update cluster API, try to HScale without backup", errDetail)
	} else {
		targetBackup = getTargetBackup(adminBackups)
	}

	// Helper to execute scale operation based on client type
	executeScale := func(opsBody admin.OpsHScale) (*http.Response, error) {
		if r.client.IsAdminClient() {
			_, resp, err := admin.NewOpsrequestApi(r.client.AdminClient()).HorizontalScaleCluster(
				r.client.AdminCtx(), data.OrgName.ValueString(), data.Name.ValueString(), opsBody)
			return resp, err
		}

		var kbBody kbcloud.OpsHScale
		b, _ := json.Marshal(opsBody)
		_ = json.Unmarshal(b, &kbBody)
		_, resp, err := kbcloud.NewOpsrequestApi(r.client.Client()).HorizontalScaleCluster(
			r.client.Ctx(), data.OrgName.ValueString(), data.Name.ValueString(), kbBody)
		return resp, err
	}

	opsBody := admin.OpsHScale{Component: update.Component.ValueString()}
	if targetBackup != "" {
		opsBody.BackupName.Set(pointer.String(targetBackup))
	}

	if shardChanged && replicasChanged {
		// Shard first
		opsBodyShard := opsBody
		opsBodyShard.Shards.Set(pointer.Int32(int32(update.CompNum.ValueInt64())))
		apiResp, err = executeScale(opsBodyShard)
		if err != nil {
			errDetail := utils.GetRespErrorDetail(apiResp)
			diags.AddError("update cluster failed when horizontal scale shards", errDetail)
			return diags
		}

		// Replicas second
		opsBodyReplicas := opsBody
		opsBodyReplicas.Replicas.Set(pointer.Int32(int32(update.Replicas.ValueInt64())))
		apiResp, err = executeScale(opsBodyReplicas)
	} else {
		if shardChanged {
			opsBody.Shards.Set(pointer.Int32(int32(update.CompNum.ValueInt64())))
		}
		if replicasChanged {
			opsBody.Replicas.Set(pointer.Int32(int32(update.Replicas.ValueInt64())))
		}
		apiResp, err = executeScale(opsBody)
	}

	if err != nil {
		errDetail := utils.GetRespErrorDetail(apiResp)
		diags.AddError("update cluster failed when horizontal scale", errDetail)
	}

	return diags
}

func (r *ClusterResource) doVolumeExpand(oldComponents []admin.ComponentItem, data *mytypes.ClustersResourceModel, update *mytypes.ComponentsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	expandVolumes := make([]admin.OpsVolumeExpandVolumesItem, 0)
	for _, c := range oldComponents {
		if pointer.ValueOf(c.Component) != update.Component.ValueString() {
			continue
		}

		var updateVolumes []*mytypes.VolumesResourceModel
		if !update.Volumes.IsNull() && !update.Volumes.IsUnknown() {
			diags.Append(update.Volumes.ElementsAs(context.Background(), &updateVolumes, false)...)
			if diags.HasError() {
				return diags
			}
		}

		for _, v := range c.Volumes {
			for _, u := range updateVolumes {
				if pointer.ValueOf(v.Name) == u.Name.ValueString() && u.Storage.ValueFloat64() > pointer.ValueOf(v.Storage) {
					expandVolumes = append(expandVolumes, admin.OpsVolumeExpandVolumesItem{
						Name:    pointer.ValueOf(v.Name),
						Storage: fmt.Sprintf("%dGi", cast.ToInt(u.Storage.ValueFloat64())),
					})
				}
			}
		}
	}

	if len(expandVolumes) == 0 {
		return diags
	}

	var apiResp *http.Response
	var err error

	opsBody := admin.OpsVolumeExpand{
		Component: update.Component.ValueString(),
		Volumes:   expandVolumes,
	}

	if r.client.IsAdminClient() {
		_, apiResp, err = admin.NewOpsrequestApi(r.client.AdminClient()).ClusterVolumeExpand(
			r.client.AdminCtx(),
			data.OrgName.ValueString(),
			data.Name.ValueString(),
			opsBody,
		)
	} else {
		var kbBody kbcloud.OpsVolumeExpand
		b, _ := json.Marshal(opsBody)
		_ = json.Unmarshal(b, &kbBody)
		_, apiResp, err = kbcloud.NewOpsrequestApi(r.client.Client()).ClusterVolumeExpand(
			r.client.Ctx(),
			data.OrgName.ValueString(),
			data.Name.ValueString(),
			kbBody,
		)
	}

	if err != nil {
		errDetail := utils.GetRespErrorDetail(apiResp)
		diags.AddError("update cluster failed when volume expand", errDetail)
	}

	return diags
}
