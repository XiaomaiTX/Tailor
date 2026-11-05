import { fireEvent, render, screen } from "@testing-library/react"
import type {
  PolicySelectorRulePreview,
  Relay,
} from "@/api/generated/model"
import { PolicyPage } from "./policy-page"

const relay = {
  nodeId: "relay-1",
  displayName: "West relay",
  selectable: true,
  createdBy: "admin",
  createdAt: "2026-11-05T00:00:00Z",
  updatedAt: "2026-11-05T00:00:00Z",
  device: {
    nodeId: "relay-1",
    hostname: "relay-west",
    givenName: "relay-west",
    ipv4: "100.64.0.20",
    username: "admin",
    tags: [],
    missing: false,
  },
} as Relay

function makeProps(
  selectorPreview?: PolicySelectorRulePreview,
  onPreviewSelectorRule = vi.fn(),
  onCreateSelectorRule = vi.fn()
) {
  return {
    revisions: [],
    revisionDetailLoading: false,
    devices: [],
    relays: [relay],
    audits: [],
    selectorRules: [],
    selectorRulesLoading: false,
    selectorPreview,
    selectorPreviewPending: false,
    selectorMutationPending: false,
    policyWritesEnabled: true,
    isLoading: false,
    t: (key: string) => key,
    locale: "en-US",
    onRollback: vi.fn(),
    onViewRevision: vi.fn(),
    onPreviewSelectorRule,
    onCreateSelectorRule,
    onDeleteSelectorRule: vi.fn(),
  }
}

describe("PolicyPage selector rule editor", () => {
  it("previews the current scope before submitting the original selector", () => {
    const onPreviewSelectorRule = vi.fn()
    const onCreateSelectorRule = vi.fn()
    const props = makeProps(undefined, onPreviewSelectorRule, onCreateSelectorRule)
    const { rerender } = render(<PolicyPage {...props} />)

    fireEvent.change(screen.getByLabelText("relay.policy.sourceSelector"), {
      target: { value: "group:admins" },
    })
    fireEvent.change(screen.getByLabelText("relay.policy.targetRelay"), {
      target: { value: "relay-1" },
    })
    fireEvent.click(screen.getByRole("button", { name: "relay.policy.previewScope" }))

    expect(onPreviewSelectorRule).toHaveBeenCalledWith({
      sourceSelector: "group:admins",
      relayNodeId: "relay-1",
    })

    rerender(
      <PolicyPage
        {...props}
        selectorPreview={{
          sourceSelector: "group:admins",
          relayNodeId: "relay-1",
          relaySelector: "100.64.0.20",
          relay,
          sourceDevices: [],
          basePolicyHash: "a".repeat(64),
          scopeHash: "b".repeat(64),
          dynamic: true,
          warning: "future members are included",
        }}
      />
    )

    fireEvent.click(
      screen.getByRole("button", { name: "relay.policy.createSelectorRule" })
    )
    expect(onCreateSelectorRule).toHaveBeenCalledWith({
      sourceSelector: "group:admins",
      relayNodeId: "relay-1",
      basePolicyHash: "a".repeat(64),
      scopeHash: "b".repeat(64),
    })
  })
})
