/* eslint-disable */
/* prettier-ignore */
// @ts-nocheck
// Generated by unplugin-vue-router. ‼️ DO NOT MODIFY THIS FILE ‼️
// It's recommended to commit this file.
// Make sure to add this file to your tsconfig.json file as an "includes" or "files" entry.

declare module 'vue-router/auto-routes' {
  import type {
    RouteRecordInfo,
    ParamValue,
    ParamValueOneOrMore,
    ParamValueZeroOrMore,
    ParamValueZeroOrOne,
  } from 'unplugin-vue-router/types'

  /**
   * Route name map generated by unplugin-vue-router
   */
  export interface RouteNamedMap {
    '/': RouteRecordInfo<'/', '/', Record<never, never>, Record<never, never>>,
    '/[...all]': RouteRecordInfo<'/[...all]', '/:all(.*)', { all: ParamValue<true> }, { all: ParamValue<false> }>,
    '/container/[id]': RouteRecordInfo<'/container/[id]', '/container/:id', { id: ParamValue<true> }, { id: ParamValue<false> }>,
    '/group/[name]': RouteRecordInfo<'/group/[name]', '/group/:name', { name: ParamValue<true> }, { name: ParamValue<false> }>,
    '/login': RouteRecordInfo<'/login', '/login', Record<never, never>, Record<never, never>>,
    '/merged/[name]': RouteRecordInfo<'/merged/[name]', '/merged/:name', { name: ParamValue<true> }, { name: ParamValue<false> }>,
    '/service/[name]': RouteRecordInfo<'/service/[name]', '/service/:name', { name: ParamValue<true> }, { name: ParamValue<false> }>,
    '/settings': RouteRecordInfo<'/settings', '/settings', Record<never, never>, Record<never, never>>,
    '/show': RouteRecordInfo<'/show', '/show', Record<never, never>, Record<never, never>>,
    '/stack/[name]': RouteRecordInfo<'/stack/[name]', '/stack/:name', { name: ParamValue<true> }, { name: ParamValue<false> }>,
  }
}