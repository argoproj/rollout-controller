/**
 * pkg/apiclient/rollout/rollout.proto
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * OpenAPI spec version: version not set
 * 
 *
 * NOTE: This file is auto generated by the swagger code generator program.
 * https://github.com/swagger-api/swagger-codegen.git
 * Do not edit the file manually.
 */

import * as api from "./api"
import { Configuration } from "./configuration"

const config: Configuration = {}

describe("RolloutServiceApi", () => {
  let instance: api.RolloutServiceApi
  beforeEach(function() {
    instance = new api.RolloutServiceApi(config)
  });

  test("rolloutServiceAbortRollout", () => {
    const body: api.RolloutAbortRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServiceAbortRollout(body, namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceGetNamespace", () => {
    return expect(instance.rolloutServiceGetNamespace({})).resolves.toBe(null)
  })
  test("rolloutServiceGetRolloutInfo", () => {
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServiceGetRolloutInfo(namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceListRolloutInfos", () => {
    const namespace: string = "namespace_example"
    return expect(instance.rolloutServiceListRolloutInfos(namespace, {})).resolves.toBe(null)
  })
  test("rolloutServicePromoteFullRollout", () => {
    const body: api.RolloutPromoteFullRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServicePromoteFullRollout(body, namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServicePromoteRollout", () => {
    const body: api.RolloutPromoteRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServicePromoteRollout(body, namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceRestartRollout", () => {
    const body: api.RolloutRestartRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServiceRestartRollout(body, namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceRetryRollout", () => {
    const body: api.RolloutRetryRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServiceRetryRollout(body, namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceSetRolloutImage", () => {
    const body: api.RolloutSetImageRequest = undefined
    const namespace: string = "namespace_example"
    const rollout: string = "rollout_example"
    const container: string = "container_example"
    const image: string = "image_example"
    const tag: string = "tag_example"
    return expect(instance.rolloutServiceSetRolloutImage(body, namespace, rollout, container, image, tag, {})).resolves.toBe(null)
  })
  test("rolloutServiceUndoRollout", () => {
    const body: api.RolloutUndoRolloutRequest = undefined
    const namespace: string = "namespace_example"
    const rollout: string = "rollout_example"
    const revision: string = "revision_example"
    return expect(instance.rolloutServiceUndoRollout(body, namespace, rollout, revision, {})).resolves.toBe(null)
  })
  test("rolloutServiceVersion", () => {
    return expect(instance.rolloutServiceVersion({})).resolves.toBe(null)
  })
  test("rolloutServiceWatchRolloutInfo", () => {
    const namespace: string = "namespace_example"
    const name: string = "name_example"
    return expect(instance.rolloutServiceWatchRolloutInfo(namespace, name, {})).resolves.toBe(null)
  })
  test("rolloutServiceWatchRolloutInfos", () => {
    const namespace: string = "namespace_example"
    return expect(instance.rolloutServiceWatchRolloutInfos(namespace, {})).resolves.toBe(null)
  })
})

