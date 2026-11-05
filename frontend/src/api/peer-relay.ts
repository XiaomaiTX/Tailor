import { customFetch } from "@/api/http"
import {
  cancelApplication,
  cancelRelay,
  createApplication,
  getCanvasGraph,
  getGraph,
  getPeerRelaySettings,
  getPolicyRevisionDetail,
  getRelayDashboard,
  getRelayMe,
  listApplications,
  listAuditLogs,
  listDevices,
  listPolicyRevisions,
  listRelays,
  registerRelay,
  reviewApplication,
  rollbackPolicyRevision,
  syncHeadscale,
  updateAssignments,
  updateAssignmentsBatch,
  updatePeerRelaySettings,
  updateRelay,
} from "@/api/generated/client"
import type {
  AssignmentInput,
  AssignmentResult,
  BatchAssignmentInput,
  BatchAssignmentResult,
  AuditLog,
  CanvasGraph,
  Device,
  GetGraphParams,
  ListApplicationsParams,
  ListDevicesParams,
  ListRelaysParams,
  PolicyRevision,
  PolicyRevisionDetail,
  PeerRelaySettings,
  PeerRelaySettingsInput,
  Relay,
  RelayApplication,
  RelayApplicationInput,
  RelayDashboard,
  RelayGraph,
  RelayRegistration,
  RelayUpdate,
  RelayUser,
  ReviewInput,
} from "@/api/generated/model"

export const peerRelayApi = {
  async logout() {
    await customFetch<void>("/auth/logout", { method: "POST" })
  },
  async me() {
    return (await getRelayMe()).data as RelayUser
  },
  async dashboard() {
    return (await getRelayDashboard()).data as RelayDashboard
  },
  async settings() {
    return (await getPeerRelaySettings()).data as PeerRelaySettings
  },
  async saveSettings(input: PeerRelaySettingsInput) {
    return (await updatePeerRelaySettings(input)).data as PeerRelaySettings
  },
  async devices(search = "") {
    const params: ListDevicesParams = search ? { search } : {}
    return (await listDevices(params)).data as Device[]
  },
  async relays(includeAll = false) {
    const params: ListRelaysParams = includeAll ? { includeAll } : {}
    return (await listRelays(params)).data as Relay[]
  },
  async graph(sourceNodeId: string) {
    const params: GetGraphParams = { sourceNodeId }
    return (await getGraph(params)).data as RelayGraph
  },
  async canvasGraph() {
    return (await getCanvasGraph()).data as CanvasGraph
  },
  async applications(status?: ListApplicationsParams["status"]) {
    const params: ListApplicationsParams = status ? { status } : {}
    return (await listApplications(params)).data as RelayApplication[]
  },
  async revisions() {
    return (await listPolicyRevisions()).data as PolicyRevision[]
  },
  async revisionDetail(id: string) {
    return (await getPolicyRevisionDetail(id)).data as PolicyRevisionDetail
  },
  async audits() {
    return (await listAuditLogs()).data as AuditLog[]
  },
  async saveAssignments(input: AssignmentInput) {
    return (await updateAssignments(input)).data as AssignmentResult
  },
  async saveAssignmentsBatch(input: BatchAssignmentInput) {
    return (await updateAssignmentsBatch(input)).data as BatchAssignmentResult
  },
  async apply(input: RelayApplicationInput) {
    return (await createApplication(input)).data as RelayApplication
  },
  async cancelApplication(id: string) {
    await cancelApplication(id)
  },
  async review(id: string, input: ReviewInput) {
    return (await reviewApplication(id, input)).data as RelayApplication
  },
  async register(input: RelayRegistration) {
    return (await registerRelay(input)).data as Relay
  },
  async updateRelay(nodeId: string, input: RelayUpdate) {
    return (await updateRelay(nodeId, input)).data as Relay
  },
  async cancelRelay(nodeId: string) {
    await cancelRelay(nodeId)
  },
  async sync() {
    await syncHeadscale()
  },
  async rollback(id: string) {
    return (await rollbackPolicyRevision(id)).data as PolicyRevision
  },
}
