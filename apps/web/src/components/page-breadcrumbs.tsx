import { Fragment, type ReactElement } from 'react'
import { Link } from 'react-router'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@escalite/ui'
import { HomeIcon } from 'lucide-react'

import { t } from '../lib/i18n'

export type PageBreadcrumbItem = {
  label: string
  href?: string
}

type PageBreadcrumbsProps = {
  items: PageBreadcrumbItem[]
}

export function PageBreadcrumbs({ items }: PageBreadcrumbsProps): ReactElement {
  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink
            aria-label={t('nav.dashboard')}
            render={<Link to="/dashboard" />}
          >
            <HomeIcon aria-hidden="true" className="size-4" />
          </BreadcrumbLink>
        </BreadcrumbItem>
        {items.map((item, index) => (
          <Fragment key={`${item.label}-${index}`}>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              {item.href ? (
                <BreadcrumbLink render={<Link to={item.href} />}>{item.label}</BreadcrumbLink>
              ) : (
                <BreadcrumbPage>{item.label}</BreadcrumbPage>
              )}
            </BreadcrumbItem>
          </Fragment>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
