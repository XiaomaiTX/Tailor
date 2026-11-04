import { ApiProblemError, customFetch } from "@/api/http"

describe("customFetch", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("拼接 API 基址并补充 JSON 请求头", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ status: "ok" }), { status: 200 })
      )
    vi.stubGlobal("fetch", fetchMock)

    await customFetch("/probe", { method: "POST", body: JSON.stringify({}) })

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/probe",
      expect.objectContaining({
        headers: expect.any(Headers),
        method: "POST",
      })
    )
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get("Accept")).toBe("application/json")
    expect(headers.get("Content-Type")).toBe("application/json")
  })

  it("将 Problem Details 转换为类型化错误", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            type: "urn:go-react-starter:problem:validation",
            title: "Bad Request",
            status: 400,
          }),
          {
            status: 400,
            headers: { "Content-Type": "application/problem+json" },
          }
        )
      )
    )
    await expect(customFetch("/probe")).rejects.toBeInstanceOf(ApiProblemError)
  })
})
