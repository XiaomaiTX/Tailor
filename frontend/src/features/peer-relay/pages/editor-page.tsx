import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import {
  applyNodeChanges,
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeChange,
  type NodeProps,
  type Viewport,
} from "@xyflow/react"
import { useBlocker } from "react-router-dom"
import "@xyflow/react/dist/style.css"
import {
  Check,
  GitFork,
  LockKeyhole,
  Plus,
  RadioTower,
  Search,
  Server,
  SlidersHorizontal,
  TriangleAlert,
  X,
} from "lucide-react"
import type {
  BatchAssignmentInput,
  CanvasEdge,
  CanvasGraph,
  CanvasNode,
} from "@/api/generated/model"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  EmptyState,
  LoadingState,
  Notice,
} from "@/features/peer-relay/components/view-primitives"
import { errorMessage } from "@/features/peer-relay/utils"
import { useI18n } from "@/i18n"
import { cn } from "@/lib/utils"

const layoutVersion = 2
const gridWidth = 320
const gridHeight = 216
const relayColumnX = 0
const deviceColumnX = gridWidth * 2
const snapGrid: [number, number] = [24, 24]
const EMPTY_CANVAS_NODES: CanvasNode[] = []

type NodeData = {
  canvasNode: CanvasNode
  focused: boolean
}
type DeviceFlowNode = Node<NodeData, "device">
type RelationData = {
  sourceKind: string
  editable: boolean
  sourceNodeId: string
  relayNodeId: string
  warning?: string
  grantMatches?: CanvasEdge["grantMatches"]
}
type DeviceFlowEdge = Edge<RelationData>
type StoredLayout = {
  version: number
  positions: Record<string, { x: number; y: number }>
  viewport?: Viewport
}

const sourceColors: Record<string, string> = {
  platform: "var(--primary)",
  new: "var(--primary)",
  external: "var(--muted-foreground)",
  inherited: "var(--chart-2)",
  unknown: "var(--chart-4)",
  exception: "var(--destructive)",
}

function deviceName(node: CanvasNode) {
  return node.device.givenName || node.device.hostname
}

function defaultPositions(
  nodes: CanvasNode[],
  linkedNodeIDs: ReadonlySet<string>
) {
  const positions: Record<string, { x: number; y: number }> = {}
  for (const [columnX, columnNodes] of [
    [relayColumnX, nodes.filter((node) => node.relay)],
    [deviceColumnX, nodes.filter((node) => !node.relay)],
  ] as const) {
    const sorted = [...columnNodes].sort((left, right) => {
      const connectionOrder =
        Number(linkedNodeIDs.has(right.device.nodeId)) -
        Number(linkedNodeIDs.has(left.device.nodeId))
      return (
        connectionOrder ||
        deviceName(left).localeCompare(deviceName(right)) ||
        left.device.nodeId.localeCompare(right.device.nodeId)
      )
    })
    sorted.forEach((node, index) => {
      positions[node.device.nodeId] = {
        x: columnX,
        y: index * gridHeight,
      }
    })
  }
  return positions
}

function addMissingPositions(
  nodes: CanvasNode[],
  current: Record<string, { x: number; y: number }>,
  linkedNodeIDs: ReadonlySet<string>
) {
  const next = { ...current }
  const liveIDs = new Set(nodes.map((node) => node.device.nodeId))
  for (const nodeID of Object.keys(next)) {
    if (!liveIDs.has(nodeID)) delete next[nodeID]
  }
  const occupied = new Set(
    Object.values(next).map(
      (position) =>
        `${Math.round(position.x / gridWidth)}:${Math.round(position.y / gridHeight)}`
    )
  )
  const missing = nodes
    .filter((node) => !next[node.device.nodeId])
    .sort((left, right) => {
      const connectionOrder =
        Number(linkedNodeIDs.has(right.device.nodeId)) -
        Number(linkedNodeIDs.has(left.device.nodeId))
      return (
        connectionOrder ||
        deviceName(left).localeCompare(deviceName(right)) ||
        left.device.nodeId.localeCompare(right.device.nodeId)
      )
    })
  for (const node of missing) {
    const columnX = node.relay ? relayColumnX : deviceColumnX
    const column = Math.round(columnX / gridWidth)
    let row = 0
    while (occupied.has(`${column}:${row}`)) row++
    const cell = `${column}:${row}`
    occupied.add(cell)
    next[node.device.nodeId] = {
      x: columnX,
      y: row * gridHeight,
    }
  }
  return next
}

function reconcileFlowNodes(
  currentNodes: DeviceFlowNode[],
  nextNodes: DeviceFlowNode[]
) {
  const currentByID = new Map(currentNodes.map((node) => [node.id, node]))
  let changed = currentNodes.length !== nextNodes.length
  const reconciled = nextNodes.map((nextNode) => {
    const currentNode = currentByID.get(nextNode.id)
    if (!currentNode) {
      changed = true
      return nextNode
    }

    const isUnchanged =
      currentNode.type === nextNode.type &&
      currentNode.position.x === nextNode.position.x &&
      currentNode.position.y === nextNode.position.y &&
      currentNode.data === nextNode.data &&
      currentNode.selected === nextNode.selected &&
      currentNode.draggable === nextNode.draggable &&
      currentNode.connectable === nextNode.connectable &&
      currentNode.deletable === nextNode.deletable
    if (isUnchanged) return currentNode

    changed = true
    return { ...currentNode, ...nextNode }
  })

  return changed ? reconciled : currentNodes
}

function readLayout(key: string): StoredLayout | null {
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredLayout
    if (
      parsed.version !== layoutVersion ||
      !parsed.positions ||
      typeof parsed.positions !== "object" ||
      Array.isArray(parsed.positions)
    ) {
      return null
    }
    const positions: StoredLayout["positions"] = {}
    for (const [nodeID, position] of Object.entries(parsed.positions)) {
      if (
        position &&
        Number.isFinite(position.x) &&
        Number.isFinite(position.y)
      ) {
        positions[nodeID] = { x: position.x, y: position.y }
      }
    }
    const viewport = parsed.viewport
    const validViewport =
      viewport &&
      Number.isFinite(viewport.x) &&
      Number.isFinite(viewport.y) &&
      Number.isFinite(viewport.zoom) &&
      viewport.zoom > 0
        ? viewport
        : undefined
    return { version: layoutVersion, positions, viewport: validViewport }
  } catch {
    return null
  }
}

const DeviceCanvasNode = memo(
  function DeviceCanvasNode({ data, selected }: NodeProps<DeviceFlowNode>) {
    const { t } = useI18n()
    const { canvasNode } = data
    const device = canvasNode.device
    const isRelay = Boolean(canvasNode.relay)
    const online = device.online
    return (
      <div
        className={cn(
          "relative w-[260px] border bg-card px-3 py-3 shadow-sm transition-colors",
          selected ? "border-primary ring-1 ring-primary" : "border-border",
          data.focused ? "opacity-100" : "opacity-50",
          device.missing && "border-dashed opacity-70"
        )}
      >
        <Handle
          type="target"
          position={Position.Left}
          isConnectable={Boolean(canvasNode.canEdit && canvasNode.canConfirm)}
          className="!size-2.5 !border-2 !border-background !bg-primary"
        />
        <div className="flex items-start gap-2.5">
          <div
            className={cn(
              "mt-0.5 flex size-8 shrink-0 items-center justify-center border",
              isRelay
                ? "border-primary/40 bg-primary/10 text-primary"
                : "border-border bg-muted text-muted-foreground"
            )}
          >
            {isRelay ? <RadioTower size={15} /> : <Server size={15} />}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <p className="truncate text-xs font-semibold">
                {deviceName(canvasNode)}
              </p>
              {isRelay ? (
                <span className="shrink-0 border border-primary/30 px-1.5 py-0.5 text-[8px] font-bold tracking-wide text-primary uppercase">
                  {t("relay.editor.relayRole")}
                </span>
              ) : null}
            </div>
            <p className="mt-1 truncate font-mono text-[9px] text-muted-foreground">
              {device.ipv4 || device.ipv6 || "—"}
            </p>
            <p className="mt-1 truncate text-[9px] text-muted-foreground">
              {device.username}
            </p>
          </div>
        </div>
        <div className="mt-3 flex items-center justify-between gap-2 border-t border-border pt-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <span
              className={cn(
                "size-1.5 shrink-0 rounded-full",
                online
                  ? "bg-primary"
                  : online === false
                    ? "bg-muted-foreground/50"
                    : "bg-muted-foreground/30"
              )}
            />
            <span className="truncate text-[9px] text-muted-foreground">
              {device.missing
                ? t("relay.editor.missing")
                : online
                  ? t("relay.common.online")
                  : online === false
                    ? t("relay.common.offline")
                    : t("relay.common.unknown")}
            </span>
            {device.tags.slice(0, 2).map((tag) => (
              <span
                key={tag}
                className="max-w-20 truncate border border-border px-1 text-[8px] text-muted-foreground"
              >
                {tag}
              </span>
            ))}
          </div>
          {isRelay ? (
            <span
              className={cn(
                "shrink-0 text-[9px] font-medium",
                canvasNode.relay?.selectable
                  ? "text-primary"
                  : "text-muted-foreground"
              )}
            >
              {canvasNode.relay?.selectable
                ? t("relay.relays.statusOpen")
                : t("relay.relays.statusPaused")}
            </span>
          ) : null}
        </div>
        <Handle
          type="source"
          position={Position.Right}
          isConnectable={Boolean(
            isRelay && canvasNode.canConfirm && canvasNode.relay?.selectable
          )}
          className="!size-2.5 !border-2 !border-background !bg-primary"
        />
      </div>
    )
  },
  (previous, next) =>
    previous.selected === next.selected &&
    previous.data.canvasNode === next.data.canvasNode &&
    previous.data.focused === next.data.focused
)

const nodeTypes = { device: DeviceCanvasNode }

function edgeSource(edge: CanvasEdge): string {
  return edge.source
}

function relationPair(sourceNodeID: string, relayNodeID: string) {
  return `${sourceNodeID}\u0000${relayNodeID}`
}

function sourceLabel(source: string, t: (key: string) => string) {
  return t(`relay.editor.${source || "new"}`)
}

function formatMessage(
  key: string,
  t: (key: string) => string,
  values: Record<string, string | number>
) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  )
}

function AuthorizationCanvas({
  graph,
  username,
  initialNodeID,
  t,
  isLoading,
  loadError,
  savePending,
  saveError,
  resetKey,
  onReload,
  onSave,
}: {
  graph?: CanvasGraph
  username: string
  initialNodeID: string
  t: (key: string) => string
  isLoading: boolean
  loadError: Error | null
  savePending: boolean
  saveError: Error | null
  resetKey: number
  onReload: () => void
  onSave: (input: BatchAssignmentInput) => void
}) {
  const layoutKey = `tailor.authorization-canvas.v${layoutVersion}:${encodeURIComponent(username)}`
  const [positions, setPositions] = useState<
    Record<string, { x: number; y: number }>
  >(() => readLayout(layoutKey)?.positions ?? {})
  const [viewport, setViewport] = useState<Viewport>(
    () => readLayout(layoutKey)?.viewport ?? { x: 0, y: 0, zoom: 1 }
  )
  const [search, setSearch] = useState("")
  const [tagFilter, setTagFilter] = useState("")
  const [relayFilter, setRelayFilter] = useState("all")
  const [relationshipFilter, setRelationshipFilter] = useState("linked")
  const [sourceFilter, setSourceFilter] = useState("all")
  const [selectedNodeID, setSelectedNodeID] = useState(initialNodeID)
  const [controlledFlowNodes, setControlledFlowNodes] = useState<
    DeviceFlowNode[]
  >([])
  const [draftTargets, setDraftTargets] = useState<Record<
    string,
    string[]
  > | null>(null)
  const [relayChoice, setRelayChoice] = useState("")
  const [showSaveConfirm, setShowSaveConfirm] = useState(false)
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false)
  const previousResetKey = useRef(resetKey)
  const { fitView } = useReactFlow()
  const nodesFromGraph = graph?.nodes ?? EMPTY_CANVAS_NODES
  useEffect(() => {
    if (previousResetKey.current === resetKey) return
    previousResetKey.current = resetKey
    setDraftTargets(null)
    setShowSaveConfirm(false)
    setShowDiscardConfirm(false)
  }, [resetKey])

  const baselineTargets = useMemo(() => {
    const result: Record<string, string[]> = {}
    for (const edge of graph?.edges ?? []) {
      result[edge.sourceNodeId] ??= []
      result[edge.sourceNodeId].push(edge.relayNodeId)
    }
    for (const targets of Object.values(result)) targets.sort()
    return result
  }, [graph])
  const currentTargets = draftTargets ?? baselineTargets
  const linkedNodeIDs = useMemo(() => {
    const result = new Set<string>()
    for (const [sourceNodeID, relayNodeIDs] of Object.entries(currentTargets)) {
      if (relayNodeIDs.length > 0) result.add(sourceNodeID)
      for (const relayNodeID of relayNodeIDs) result.add(relayNodeID)
    }
    return result
  }, [currentTargets])
  const effectivePositions = useMemo(
    () => addMissingPositions(nodesFromGraph, positions, linkedNodeIDs),
    [linkedNodeIDs, nodesFromGraph, positions]
  )

  useEffect(() => {
    if (!graph) return
    const timeout = window.setTimeout(() => {
      try {
        window.localStorage.setItem(
          layoutKey,
          JSON.stringify({
            version: layoutVersion,
            positions: effectivePositions,
            viewport,
          } satisfies StoredLayout)
        )
      } catch {
        // Layout persistence is optional when browser storage is unavailable.
      }
    }, 250)
    return () => window.clearTimeout(timeout)
  }, [effectivePositions, graph, layoutKey, viewport])

  const targetIDs = useCallback(
    (sourceNodeID: string) =>
      currentTargets[sourceNodeID] ?? baselineTargets[sourceNodeID] ?? [],
    [baselineTargets, currentTargets]
  )
  const currentPairKeys = useMemo(() => {
    const pairs = new Set<string>()
    for (const [sourceNodeID, relayNodeIDs] of Object.entries(currentTargets)) {
      for (const relayNodeID of relayNodeIDs) {
        pairs.add(relationPair(sourceNodeID, relayNodeID))
      }
    }
    return pairs
  }, [currentTargets])
  const changedSourceIDs = useMemo(() => {
    if (!graph || !draftTargets) return []
    const sourceIDs = new Set([
      ...Object.keys(baselineTargets),
      ...Object.keys(draftTargets),
      ...graph.nodes
        .filter((node) => node.canEdit)
        .map((node) => node.device.nodeId),
    ])
    return [...sourceIDs]
      .filter((sourceID) => {
        const before = [...(baselineTargets[sourceID] ?? [])].sort()
        const after = [
          ...(draftTargets[sourceID] ?? baselineTargets[sourceID] ?? []),
        ].sort()
        return (
          before.length !== after.length ||
          before.some((id, index) => id !== after[index])
        )
      })
      .sort()
  }, [baselineTargets, draftTargets, graph])
  const dirty = changedSourceIDs.length > 0

  const addOrRemoveTarget = useCallback(
    (sourceNodeID: string, relayNodeID: string, add: boolean) => {
      setDraftTargets((current) => {
        const starting = current ?? baselineTargets
        const targets = [
          ...(starting[sourceNodeID] ?? baselineTargets[sourceNodeID] ?? []),
        ]
        const nextTargets = add
          ? [...new Set([...targets, relayNodeID])]
          : targets.filter((id) => id !== relayNodeID)
        return { ...starting, [sourceNodeID]: nextTargets.sort() }
      })
    },
    [baselineTargets]
  )

  const canConnect = useCallback(
    (connection: Connection | Edge) => {
      if (loadError || !graph || !connection.source || !connection.target)
        return false
      const source = graph.nodes.find(
        (node) => node.device.nodeId === connection.source
      )
      const target = graph.nodes.find(
        (node) => node.device.nodeId === connection.target
      )
      if (
        !source?.relay?.selectable ||
        !source.canConfirm ||
        !target?.canEdit ||
        !target.canConfirm ||
        connection.source === connection.target
      ) {
        return false
      }
      return !(
        currentTargets[connection.target] ??
        baselineTargets[connection.target] ??
        []
      ).includes(connection.source)
    },
    [baselineTargets, currentTargets, graph, loadError]
  )

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!canConnect(connection) || !connection.source || !connection.target)
        return
      addOrRemoveTarget(connection.target, connection.source, true)
    },
    [addOrRemoveTarget, canConnect]
  )

  const onEdgesDelete = useCallback(
    (removedEdges: DeviceFlowEdge[]) => {
      for (const edge of removedEdges) {
        if (edge.data?.editable) {
          addOrRemoveTarget(
            edge.data.sourceNodeId,
            edge.data.relayNodeId,
            false
          )
        }
      }
    },
    [addOrRemoveTarget]
  )

  const normalizedSearch = search.trim().toLocaleLowerCase()
  const visibleNodes = useMemo(() => {
    const relationSources = new Map<string, Set<string>>()
    const presentPairs = new Set<string>()
    for (const edge of graph?.edges ?? []) {
      const pair = relationPair(edge.sourceNodeId, edge.relayNodeId)
      if (!currentPairKeys.has(pair)) continue
      presentPairs.add(pair)
      const sources =
        relationSources.get(edge.sourceNodeId) ?? new Set<string>()
      sources.add(edgeSource(edge))
      relationSources.set(edge.sourceNodeId, sources)
      const targetSources =
        relationSources.get(edge.relayNodeId) ?? new Set<string>()
      targetSources.add(edgeSource(edge))
      relationSources.set(edge.relayNodeId, targetSources)
    }
    for (const [sourceID, targets] of Object.entries(currentTargets)) {
      for (const relayID of targets) {
        if (presentPairs.has(relationPair(sourceID, relayID))) continue
        const sources = relationSources.get(sourceID) ?? new Set<string>()
        sources.add("new")
        relationSources.set(sourceID, sources)
        const targetSources = relationSources.get(relayID) ?? new Set<string>()
        targetSources.add("new")
        relationSources.set(relayID, targetSources)
      }
    }
    return nodesFromGraph.filter((node) => {
      const text = [
        node.device.givenName,
        node.device.hostname,
        node.device.username,
        node.device.ipv4,
        node.device.ipv6,
        node.device.tags.join(" "),
        node.relay?.displayName,
        node.relay?.region,
      ]
        .filter(Boolean)
        .join(" ")
        .toLocaleLowerCase()
      if (normalizedSearch && !text.includes(normalizedSearch)) return false
      if (tagFilter && !node.device.tags.includes(tagFilter)) return false
      if (relayFilter === "relay" && !node.relay) return false
      if (relayFilter === "device" && node.relay) return false
      const sources =
        relationSources.get(node.device.nodeId) ?? new Set<string>()
      if (relationshipFilter === "linked" && sources.size === 0) return false
      if (relationshipFilter === "unlinked" && sources.size > 0) return false
      if (sourceFilter !== "all" && !sources.has(sourceFilter)) return false
      return true
    })
  }, [
    currentTargets,
    currentPairKeys,
    graph?.edges,
    nodesFromGraph,
    normalizedSearch,
    relayFilter,
    relationshipFilter,
    sourceFilter,
    tagFilter,
  ])

  const activeSelectedNodeID = visibleNodes.some(
    (node) => node.device.nodeId === selectedNodeID
  )
    ? selectedNodeID
    : ""

  const relatedNodeIDs = useMemo(() => {
    const related = new Set<string>()
    if (!activeSelectedNodeID) return related
    related.add(activeSelectedNodeID)
    for (const edge of graph?.edges ?? []) {
      if (edge.sourceNodeId === activeSelectedNodeID)
        related.add(edge.relayNodeId)
      if (edge.relayNodeId === activeSelectedNodeID)
        related.add(edge.sourceNodeId)
    }
    for (const [sourceNodeID, relayNodeIDs] of Object.entries(currentTargets)) {
      if (sourceNodeID === activeSelectedNodeID) {
        for (const relayNodeID of relayNodeIDs) related.add(relayNodeID)
      } else if (relayNodeIDs.includes(activeSelectedNodeID)) {
        related.add(sourceNodeID)
      }
    }
    return related
  }, [activeSelectedNodeID, currentTargets, graph?.edges])

  const flowNodeData = useMemo(() => {
    const dataByID = new Map<string, NodeData>()
    for (const canvasNode of visibleNodes) {
      dataByID.set(canvasNode.device.nodeId, {
        canvasNode,
        focused:
          !activeSelectedNodeID || relatedNodeIDs.has(canvasNode.device.nodeId),
      })
    }
    return dataByID
  }, [activeSelectedNodeID, relatedNodeIDs, visibleNodes])
  const flowNodeInputs = useMemo<DeviceFlowNode[]>(
    () =>
      visibleNodes.map((canvasNode) => {
        const id = canvasNode.device.nodeId
        return {
          id,
          type: "device",
          position: effectivePositions[id] ?? { x: 0, y: 0 },
          data: flowNodeData.get(id)!,
          selected: activeSelectedNodeID === id,
          draggable: true,
          connectable: Boolean(
            (canvasNode.canEdit && canvasNode.canConfirm) ||
            (canvasNode.relay?.selectable && canvasNode.canConfirm)
          ),
          deletable: false,
        }
      }),
    [effectivePositions, activeSelectedNodeID, flowNodeData, visibleNodes]
  )

  const flowNodes = useMemo(
    () => reconcileFlowNodes(controlledFlowNodes, flowNodeInputs),
    [controlledFlowNodes, flowNodeInputs]
  )
  const flowNodeInputsRef = useRef(flowNodeInputs)
  useLayoutEffect(() => {
    flowNodeInputsRef.current = flowNodeInputs
  }, [flowNodeInputs])

  const flowEdges = useMemo<DeviceFlowEdge[]>(() => {
    const visibleIDs = new Set(visibleNodes.map((node) => node.device.nodeId))
    const existingPairs = new Set<string>()
    const edges: DeviceFlowEdge[] = []
    for (const edge of graph?.edges ?? []) {
      if (
        !visibleIDs.has(edge.sourceNodeId) ||
        !visibleIDs.has(edge.relayNodeId)
      )
        continue
      const pair = relationPair(edge.sourceNodeId, edge.relayNodeId)
      if (!currentPairKeys.has(pair)) continue
      const id = `${edge.sourceNodeId}->${edge.relayNodeId}`
      existingPairs.add(pair)
      const focused =
        !activeSelectedNodeID ||
        edge.sourceNodeId === activeSelectedNodeID ||
        edge.relayNodeId === activeSelectedNodeID
      const color = sourceColors[edge.source] ?? sourceColors.unknown
      edges.push({
        id,
        source: edge.relayNodeId,
        target: edge.sourceNodeId,
        type: "simplebezier",
        label: sourceLabel(edge.source, t),
        labelStyle: { fill: color, fontSize: 10, fontWeight: 600 },
        labelBgStyle: { fill: "var(--card)", fillOpacity: 0.9 },
        markerEnd: { type: MarkerType.ArrowClosed, color },
        style: {
          stroke: color,
          strokeWidth: edge.source === "platform" ? 2.2 : 1.6,
          opacity: focused ? 1 : 0.14,
        },
        data: {
          sourceKind: edge.source,
          editable: edge.editable && !loadError,
          sourceNodeId: edge.sourceNodeId,
          relayNodeId: edge.relayNodeId,
          warning: edge.warning,
          grantMatches: edge.grantMatches,
        },
        deletable: edge.editable && !loadError,
        selectable: edge.editable && !loadError,
      })
    }
    for (const [sourceID, targets] of Object.entries(currentTargets)) {
      for (const relayID of targets) {
        const pair = relationPair(sourceID, relayID)
        const id = `${sourceID}->${relayID}`
        if (
          existingPairs.has(pair) ||
          !visibleIDs.has(sourceID) ||
          !visibleIDs.has(relayID)
        )
          continue
        const focused =
          !activeSelectedNodeID ||
          sourceID === activeSelectedNodeID ||
          relayID === activeSelectedNodeID
        const color = sourceColors.new
        edges.push({
          id,
          source: relayID,
          target: sourceID,
          type: "simplebezier",
          label: sourceLabel("new", t),
          labelStyle: { fill: color, fontSize: 10, fontWeight: 600 },
          labelBgStyle: { fill: "var(--card)", fillOpacity: 0.9 },
          markerEnd: { type: MarkerType.ArrowClosed, color },
          style: {
            stroke: color,
            strokeWidth: 2,
            strokeDasharray: "5 4",
            opacity: focused ? 1 : 0.14,
          },
          data: {
            sourceKind: "new",
            editable: true,
            sourceNodeId: sourceID,
            relayNodeId: relayID,
          },
          deletable: true,
          selectable: true,
        })
      }
    }
    return edges
  }, [
    currentPairKeys,
    currentTargets,
    graph?.edges,
    loadError,
    activeSelectedNodeID,
    t,
    visibleNodes,
  ])

  const onNodesChange = useCallback((changes: NodeChange<DeviceFlowNode>[]) => {
    setControlledFlowNodes((currentNodes) =>
      applyNodeChanges(
        changes,
        reconcileFlowNodes(currentNodes, flowNodeInputsRef.current)
      )
    )

    const moved = changes.filter(
      (change) => change.type === "position" && change.position
    )
    if (moved.length) {
      setPositions((current) => {
        const next = { ...current }
        for (const change of moved) {
          if (change.type === "position" && change.position) {
            next[change.id] = change.position
          }
        }
        return next
      })
    }
    for (const change of changes) {
      if (change.type === "select") {
        setSelectedNodeID((current) =>
          change.selected ? change.id : current === change.id ? "" : current
        )
      }
    }
  }, [])

  const tags = useMemo(
    () =>
      [...new Set(nodesFromGraph.flatMap((node) => node.device.tags))].sort(),
    [nodesFromGraph]
  )
  const selectedNode = graph?.nodes.find(
    (node) => node.device.nodeId === activeSelectedNodeID
  )
  const selectedTargets = selectedNode
    ? targetIDs(selectedNode.device.nodeId)
    : []
  const availableTargets = (graph?.nodes ?? []).filter(
    (node) =>
      node.relay?.selectable &&
      node.canConfirm &&
      node.device.nodeId !== activeSelectedNodeID &&
      !selectedTargets.includes(node.device.nodeId)
  )
  const relationRows = flowEdges.filter(
    (edge) =>
      edge.source === activeSelectedNodeID ||
      edge.target === activeSelectedNodeID
  )
  const addCount = changedSourceIDs.reduce((count, sourceID) => {
    const baseline = new Set(baselineTargets[sourceID] ?? [])
    return (
      count +
      (currentTargets[sourceID] ?? []).filter((id) => !baseline.has(id)).length
    )
  }, 0)
  const removeCount = changedSourceIDs.reduce((count, sourceID) => {
    const desired = new Set(currentTargets[sourceID] ?? [])
    return (
      count +
      (baselineTargets[sourceID] ?? []).filter((id) => !desired.has(id)).length
    )
  }, 0)
  const blockers = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty && currentLocation.pathname !== nextLocation.pathname
  )

  useEffect(() => {
    if (!dirty) return
    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ""
    }
    window.addEventListener("beforeunload", beforeUnload)
    return () => window.removeEventListener("beforeunload", beforeUnload)
  }, [dirty])

  const organizeLayout = () => {
    try {
      window.localStorage.removeItem(layoutKey)
    } catch {
      // The in-memory reset still works when browser storage is unavailable.
    }
    setPositions(defaultPositions(nodesFromGraph, linkedNodeIDs))
    setViewport({ x: 0, y: 0, zoom: 1 })
    window.requestAnimationFrame(() =>
      fitView({ padding: 0.18, duration: 200 })
    )
  }

  const saveChanges = () => {
    if (!graph || !dirty) return
    const changes = changedSourceIDs.map((sourceNodeId) => ({
      sourceNodeId,
      relayNodeIds: [
        ...(currentTargets[sourceNodeId] ??
          baselineTargets[sourceNodeId] ??
          []),
      ].sort(),
    }))
    setShowSaveConfirm(false)
    onSave({ changes, basePolicyHash: graph.basePolicyHash })
  }

  if (isLoading) return <LoadingState t={t} />
  if (!graph && loadError) {
    return (
      <div className="space-y-3">
        <Notice
          icon={TriangleAlert}
          tone="red"
          text={errorMessage(loadError, t("errors.unknown"))}
        />
        <Button variant="outline" size="sm" onPress={onReload}>
          {t("relay.editor.retry")}
        </Button>
      </div>
    )
  }
  if (!graph || graph.nodes.length === 0) {
    return <EmptyState title={t("relay.editor.noDevice")} icon={Server} />
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 border border-border bg-card p-4">
        <div className="flex flex-col justify-between gap-3 xl:flex-row xl:items-center">
          <div className="relative w-full xl:max-w-sm">
            <Search
              size={14}
              className="absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("relay.editor.search")}
              aria-label={t("relay.editor.search")}
              className="pl-9"
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <label className="sr-only" htmlFor="canvas-tag-filter">
              {t("relay.editor.filterTag")}
            </label>
            <select
              id="canvas-tag-filter"
              value={tagFilter}
              onChange={(event) => setTagFilter(event.target.value)}
              className="h-9 border border-input bg-background px-2 text-xs"
            >
              <option value="">{t("relay.editor.allTags")}</option>
              {tags.map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </select>
            <label className="sr-only" htmlFor="canvas-relay-filter">
              {t("relay.editor.filterRole")}
            </label>
            <select
              id="canvas-relay-filter"
              value={relayFilter}
              onChange={(event) => setRelayFilter(event.target.value)}
              className="h-9 border border-input bg-background px-2 text-xs"
            >
              <option value="all">{t("relay.editor.allRoles")}</option>
              <option value="relay">{t("relay.editor.relayRole")}</option>
              <option value="device">{t("relay.editor.deviceRole")}</option>
            </select>
            <label className="sr-only" htmlFor="canvas-relationship-filter">
              {t("relay.editor.filterRelationship")}
            </label>
            <select
              id="canvas-relationship-filter"
              value={relationshipFilter}
              onChange={(event) => setRelationshipFilter(event.target.value)}
              className="h-9 border border-input bg-background px-2 text-xs"
            >
              <option value="linked">{t("relay.editor.linkedDevices")}</option>
              <option value="unlinked">
                {t("relay.editor.unlinkedDevices")}
              </option>
              <option value="all">{t("relay.editor.allRelationships")}</option>
            </select>
            <label className="sr-only" htmlFor="canvas-source-filter">
              {t("relay.editor.filterSource")}
            </label>
            <select
              id="canvas-source-filter"
              value={sourceFilter}
              onChange={(event) => setSourceFilter(event.target.value)}
              className="h-9 border border-input bg-background px-2 text-xs"
            >
              <option value="all">{t("relay.editor.allSources")}</option>
              {[
                "platform",
                "external",
                "inherited",
                "unknown",
                "exception",
                "new",
              ].map((source) => (
                <option key={source} value={source}>
                  {sourceLabel(source, t)}
                </option>
              ))}
            </select>
            <Button
              variant="outline"
              size="sm"
              onPress={() => fitView({ padding: 0.18, duration: 200 })}
            >
              <SlidersHorizontal size={13} />
              {t("relay.editor.fitView")}
            </Button>
            <Button variant="outline" size="sm" onPress={organizeLayout}>
              <GitFork size={13} />
              {t("relay.editor.organizeLayout")}
            </Button>
          </div>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 text-[10px] text-muted-foreground">
          <span>
            {formatMessage("relay.editor.canvasCount", t, {
              count: visibleNodes.length,
              total: graph.nodes.length,
            })}
          </span>
          <span className="flex flex-wrap gap-x-3 gap-y-1">
            {["platform", "external", "inherited", "unknown", "exception"].map(
              (source) => (
                <span key={source} className="inline-flex items-center gap-1.5">
                  <span
                    className="size-2 rounded-full"
                    style={{ background: sourceColors[source] }}
                  />
                  {sourceLabel(source, t)}
                </span>
              )
            )}
          </span>
        </div>
      </div>

      {graph.policyStale ? (
        <Notice
          icon={TriangleAlert}
          tone="amber"
          text={t("relay.editor.policyUnavailable")}
        />
      ) : null}
      {loadError ? (
        <Notice
          icon={TriangleAlert}
          tone="amber"
          text={t("relay.editor.canvasRefreshFailed")}
        />
      ) : null}
      {!graph.policyWritesEnabled ? (
        <Notice
          icon={LockKeyhole}
          tone="neutral"
          text={t("relay.editor.writesDisabled")}
        />
      ) : null}
      {saveError ? (
        <Notice
          icon={TriangleAlert}
          tone="red"
          text={errorMessage(saveError, t("errors.unknown"))}
        />
      ) : null}

      {dirty ? (
        <div className="flex flex-col justify-between gap-3 border border-primary/30 bg-primary/5 p-4 sm:flex-row sm:items-center">
          <div>
            <p className="text-xs font-semibold">{t("relay.editor.summary")}</p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              {formatMessage("relay.editor.changeSummary", t, {
                added: addCount,
                removed: removeCount,
                devices: changedSourceIDs.length,
              })}
            </p>
          </div>
          <div className="flex shrink-0 gap-2">
            <Button
              variant="outline"
              size="sm"
              onPress={() => setShowDiscardConfirm(true)}
            >
              <X size={13} />
              {t("relay.editor.discard")}
            </Button>
            <Button
              size="sm"
              onPress={() => setShowSaveConfirm(true)}
              isDisabled={
                savePending ||
                !graph.policyWritesEnabled ||
                graph.policyStale ||
                Boolean(loadError)
              }
            >
              <Check size={13} />
              {t("relay.editor.reviewSave")}
            </Button>
          </div>
        </div>
      ) : null}

      <div className="grid min-h-[650px] overflow-hidden border border-border bg-card xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="relative h-[min(72vh,900px)] min-h-[560px] bg-[radial-gradient(var(--border)_1px,transparent_1px)] [background-size:22px_22px]">
          <div className="pointer-events-none absolute inset-x-4 top-3 z-10 flex justify-between text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
            <span>{t("relay.editor.relayColumn")}</span>
            <span>{t("relay.editor.deviceColumn")}</span>
          </div>
          <ReactFlow
            nodes={flowNodes}
            edges={flowEdges}
            nodeTypes={nodeTypes}
            onNodesChange={onNodesChange}
            onEdgesDelete={onEdgesDelete}
            onConnect={onConnect}
            isValidConnection={canConnect}
            onNodeClick={(_event, node) => setSelectedNodeID(node.id)}
            onPaneClick={() => setSelectedNodeID("")}
            onMoveEnd={(_event, nextViewport) => setViewport(nextViewport)}
            defaultViewport={viewport}
            snapToGrid
            snapGrid={snapGrid}
            nodesConnectable={
              graph.policyWritesEnabled && !graph.policyStale && !loadError
            }
            deleteKeyCode={["Backspace", "Delete"]}
            fitView={false}
            minZoom={0.15}
            maxZoom={2}
            connectionLineStyle={{ stroke: "var(--primary)", strokeWidth: 2 }}
            proOptions={{ hideAttribution: true }}
          >
            <Background
              variant={BackgroundVariant.Dots}
              gap={24}
              size={1}
              color="var(--border)"
            />
            <Controls showInteractive={false} />
            <MiniMap
              pannable
              zoomable
              nodeColor={(node) => {
                const data = node.data as NodeData
                return data.canvasNode.relay
                  ? "var(--primary)"
                  : "var(--muted-foreground)"
              }}
              maskColor="color-mix(in oklab, var(--background) 65%, transparent)"
              className="!border !border-border !bg-card"
            />
          </ReactFlow>
          {visibleNodes.length === 0 ? (
            <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
              <EmptyState
                title={t("relay.editor.noFilterResults")}
                icon={Search}
              />
            </div>
          ) : null}
        </div>

        <aside className="min-w-0 border-t border-border p-4 xl:border-t-0 xl:border-l">
          {selectedNode ? (
            <>
              <div className="flex items-start gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center border border-border bg-muted text-muted-foreground">
                  {selectedNode.relay ? (
                    <RadioTower size={16} />
                  ) : (
                    <Server size={16} />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-sm font-semibold">
                    {deviceName(selectedNode)}
                  </h2>
                  <p className="mt-1 truncate font-mono text-[9px] text-muted-foreground">
                    {selectedNode.device.ipv4 ||
                      selectedNode.device.ipv6 ||
                      "—"}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("relay.editor.closeDetails")}
                  onPress={() => setSelectedNodeID("")}
                >
                  <X size={14} />
                </Button>
              </div>
              <dl className="mt-5 grid grid-cols-[84px_minmax(0,1fr)] gap-x-3 gap-y-2 border-y border-border py-4 text-[10px]">
                <dt className="text-muted-foreground">
                  {t("relay.editor.owner")}
                </dt>
                <dd className="truncate">{selectedNode.device.username}</dd>
                <dt className="text-muted-foreground">
                  {t("relay.editor.status")}
                </dt>
                <dd>
                  {selectedNode.device.missing
                    ? t("relay.editor.missing")
                    : selectedNode.device.online
                      ? t("relay.common.online")
                      : selectedNode.device.online === false
                        ? t("relay.common.offline")
                        : t("relay.common.unknown")}
                </dd>
                <dt className="text-muted-foreground">
                  {t("relay.editor.tags")}
                </dt>
                <dd className="break-words">
                  {selectedNode.device.tags.join(", ") || "—"}
                </dd>
                {selectedNode.relay ? (
                  <>
                    <dt className="text-muted-foreground">
                      {t("relay.editor.relayRole")}
                    </dt>
                    <dd className="truncate">
                      {selectedNode.relay.displayName}
                    </dd>
                    <dt className="text-muted-foreground">
                      {t("relay.relays.region")}
                    </dt>
                    <dd className="truncate">
                      {selectedNode.relay.region || "—"}
                    </dd>
                    <dt className="text-muted-foreground">
                      {t("relay.editor.relayState")}
                    </dt>
                    <dd>
                      {selectedNode.relay.selectable
                        ? t("relay.relays.statusOpen")
                        : t("relay.relays.statusPaused")}
                    </dd>
                  </>
                ) : null}
              </dl>

              {selectedNode.canEdit && selectedNode.canConfirm && !loadError ? (
                <div className="mt-5">
                  <p className="text-[10px] font-bold tracking-wide text-muted-foreground">
                    {t("relay.editor.addRelation")}
                  </p>
                  <div className="mt-2 flex gap-2">
                    <select
                      aria-label={t("relay.editor.chooseTarget")}
                      value={relayChoice}
                      onChange={(event) => setRelayChoice(event.target.value)}
                      className="h-9 min-w-0 flex-1 border border-input bg-background px-2 text-xs"
                    >
                      <option value="">{t("relay.editor.chooseTarget")}</option>
                      {availableTargets.map((node) => (
                        <option
                          key={node.device.nodeId}
                          value={node.device.nodeId}
                        >
                          {node.relay?.displayName || deviceName(node)}
                        </option>
                      ))}
                    </select>
                    <Button
                      size="icon"
                      aria-label={t("relay.editor.addRelation")}
                      isDisabled={!relayChoice}
                      onPress={() => {
                        addOrRemoveTarget(
                          selectedNode.device.nodeId,
                          relayChoice,
                          true
                        )
                        setRelayChoice("")
                      }}
                    >
                      <Plus size={14} />
                    </Button>
                  </div>
                </div>
              ) : (
                <p className="mt-5 flex items-start gap-2 border border-border bg-muted/40 p-3 text-[10px] leading-4 text-muted-foreground">
                  <LockKeyhole size={13} className="mt-0.5 shrink-0" />
                  {t("relay.editor.nodeReadOnly")}
                </p>
              )}

              <div className="mt-6">
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-[10px] font-bold tracking-wide text-muted-foreground">
                    {t("relay.editor.relatedAuthorization")}
                  </p>
                  <span className="text-[10px] text-muted-foreground">
                    {relationRows.length}
                  </span>
                </div>
                {relationRows.length ? (
                  <div className="space-y-2">
                    {relationRows.map((edge) => {
                      const isOutgoing =
                        edge.data?.sourceNodeId === selectedNode.device.nodeId
                      const otherID = isOutgoing
                        ? edge.data?.relayNodeId
                        : edge.data?.sourceNodeId
                      const relatedID = otherID ?? edge.id
                      const other = graph.nodes.find(
                        (node) => node.device.nodeId === relatedID
                      )
                      const sourceKind = edge.data?.sourceKind ?? "unknown"
                      return (
                        <div
                          key={edge.id}
                          className="flex items-center gap-2 border border-border p-2.5"
                        >
                          <span
                            className="size-2 shrink-0 rounded-full"
                            style={{
                              background:
                                sourceColors[sourceKind] ??
                                sourceColors.unknown,
                            }}
                          />
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-[10px] font-semibold">
                              {other ? deviceName(other) : relatedID}
                            </p>
                            <p className="mt-0.5 truncate text-[9px] text-muted-foreground">
                              {sourceLabel(sourceKind, t)}
                              {edge.data?.warning
                                ? ` · ${edge.data.warning}`
                                : ""}
                            </p>
                            {edge.data?.grantMatches?.map((match, index) => (
                              <p
                                key={`${match.grantId}-${index}`}
                                className="mt-1 truncate text-[9px] text-muted-foreground/80"
                                title={`${match.grantId}: ${match.src.join(", ")} → ${match.dst.join(", ")}`}
                              >
                                {`${match.matchedSrc.join(", ")} → ${match.matchedDst.join(", ")} · ${match.grantId}`}
                              </p>
                            ))}
                          </div>
                          {isOutgoing && edge.data?.editable ? (
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label={formatMessage(
                                "relay.editor.removeRelation",
                                t,
                                {
                                  name: other ? deviceName(other) : relatedID,
                                }
                              )}
                              onPress={() =>
                                addOrRemoveTarget(
                                  edge.data!.sourceNodeId,
                                  edge.data!.relayNodeId,
                                  false
                                )
                              }
                            >
                              <X size={13} />
                            </Button>
                          ) : null}
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <p className="border border-dashed border-border p-3 text-[10px] leading-4 text-muted-foreground">
                    {t("relay.editor.noRelatedAuthorization")}
                  </p>
                )}
              </div>
            </>
          ) : (
            <div className="flex min-h-48 flex-col items-center justify-center text-center">
              <SlidersHorizontal size={18} className="text-muted-foreground" />
              <p className="mt-3 text-xs font-semibold">
                {t("relay.editor.selectNode")}
              </p>
              <p className="mt-1 max-w-52 text-[10px] leading-4 text-muted-foreground">
                {t("relay.editor.selectNodeHelp")}
              </p>
            </div>
          )}
          <div className="mt-6 border-t border-border pt-4">
            <p className="text-[9px] leading-4 text-muted-foreground">
              {t("relay.editor.routeNotice")}
            </p>
          </div>
        </aside>
      </div>

      {showSaveConfirm ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          role="presentation"
        >
          <section
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="canvas-save-title"
            className="w-full max-w-md border border-border bg-card p-5 shadow-xl"
          >
            <h2 id="canvas-save-title" className="text-sm font-semibold">
              {t("relay.editor.confirmSaveTitle")}
            </h2>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">
              {formatMessage("relay.editor.changeSummary", t, {
                added: addCount,
                removed: removeCount,
                devices: changedSourceIDs.length,
              })}
            </p>
            <ul className="mt-3 max-h-40 space-y-1 overflow-auto text-[10px] text-muted-foreground">
              {changedSourceIDs.map((sourceID) => {
                const node = graph.nodes.find(
                  (item) => item.device.nodeId === sourceID
                )
                return (
                  <li key={sourceID} className="truncate">
                    • {node ? deviceName(node) : sourceID}
                  </li>
                )
              })}
            </ul>
            <div className="mt-5 flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onPress={() => setShowSaveConfirm(false)}
              >
                {t("relay.common.cancel")}
              </Button>
              <Button size="sm" isPending={savePending} onPress={saveChanges}>
                {t("relay.editor.confirmSave")}
              </Button>
            </div>
          </section>
        </div>
      ) : null}

      {showDiscardConfirm ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          role="presentation"
        >
          <section
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="canvas-discard-title"
            className="w-full max-w-sm border border-border bg-card p-5 shadow-xl"
          >
            <h2 id="canvas-discard-title" className="text-sm font-semibold">
              {t("relay.editor.discardTitle")}
            </h2>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">
              {t("relay.editor.discardDescription")}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onPress={() => setShowDiscardConfirm(false)}
              >
                {t("relay.common.cancel")}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onPress={() => {
                  setDraftTargets(null)
                  setShowDiscardConfirm(false)
                }}
              >
                {t("relay.editor.discard")}
              </Button>
            </div>
          </section>
        </div>
      ) : null}

      {blockers.state === "blocked" ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          role="presentation"
        >
          <section
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="canvas-leave-title"
            className="w-full max-w-sm border border-border bg-card p-5 shadow-xl"
          >
            <h2 id="canvas-leave-title" className="text-sm font-semibold">
              {t("relay.editor.unsavedTitle")}
            </h2>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">
              {t("relay.editor.unsavedDescription")}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onPress={() => blockers.reset?.()}
              >
                {t("relay.editor.stay")}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onPress={() => blockers.proceed?.()}
              >
                {t("relay.editor.leave")}
              </Button>
            </div>
          </section>
        </div>
      ) : null}
    </div>
  )
}

export function EditorPage(props: {
  graph?: CanvasGraph
  username: string
  initialNodeID: string
  t: (key: string) => string
  isLoading: boolean
  loadError: Error | null
  savePending: boolean
  saveError: Error | null
  resetKey: number
  onReload: () => void
  onSave: (input: BatchAssignmentInput) => void
}) {
  return (
    <ReactFlowProvider>
      <AuthorizationCanvas {...props} />
    </ReactFlowProvider>
  )
}
