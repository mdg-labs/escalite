import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import {
  SIDEBAR_NAV_CONFIG,
  filterNavGroupsForRole,
  isNavItemActive,
} from '../src/lib/nav-config.ts'

describe('nav-config', () => {
  const allItems = SIDEBAR_NAV_CONFIG.flatMap((group) => group.items)

  it('includes placeholder routes for teams, schedules, and status pages', () => {
    const paths = allItems.map((item) => item.path)
    assert.ok(paths.includes('/teams'))
    assert.ok(paths.includes('/schedules'))
    assert.ok(paths.includes('/status-pages'))
  })

  it('hides admin-only items for non-admin users', () => {
    const memberGroups = filterNavGroupsForRole(false)
    const memberPaths = memberGroups.flatMap((group) => group.items.map((item) => item.path))

    assert.ok(!memberPaths.includes('/audit-log'))
    assert.ok(!memberPaths.includes('/status-pages'))
    assert.ok(!memberPaths.includes('/settings/incident-roles'))
    assert.ok(memberPaths.includes('/dashboard'))
    assert.ok(memberPaths.includes('/teams'))
  })

  it('shows admin-only items for admin users', () => {
    const adminGroups = filterNavGroupsForRole(true)
    const adminPaths = adminGroups.flatMap((group) => group.items.map((item) => item.path))

    assert.ok(adminPaths.includes('/audit-log'))
    assert.ok(adminPaths.includes('/status-pages'))
    assert.ok(adminPaths.includes('/settings/incident-roles'))
  })

  it('highlights nested routes under their parent nav item', () => {
    const services = allItems.find((item) => item.path === '/services')
    const schedules = allItems.find((item) => item.path === '/schedules')
    const alerts = allItems.find((item) => item.path === '/alerts')

    assert.equal(services && isNavItemActive('/services/abc', services), true)
    assert.equal(
      services &&
        isNavItemActive('/services/abc/escalation-policies/policy-1', services),
      true,
    )
    assert.equal(schedules && isNavItemActive('/schedules/schedule-1', schedules), true)
    assert.equal(alerts && isNavItemActive('/alerts/alert-1', alerts), true)
  })

  it('does not highlight dashboard for other routes', () => {
    const dashboard = allItems.find((item) => item.path === '/dashboard')
    assert.equal(dashboard && isNavItemActive('/alerts', dashboard), false)
  })

  it('prefers a more specific settings child over the settings parent', () => {
    const settings = allItems.find((item) => item.path === '/settings')
    const roleDefinitions = allItems.find((item) => item.path === '/settings/incident-roles')

    assert.equal(settings && isNavItemActive('/settings/notifications', settings), true)
    assert.equal(settings && isNavItemActive('/settings/incident-roles', settings), false)
    assert.equal(
      roleDefinitions && isNavItemActive('/settings/incident-roles', roleDefinitions),
      true,
    )
  })
})
