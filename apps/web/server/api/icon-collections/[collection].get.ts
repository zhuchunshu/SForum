import { createError, getQuery } from 'h3'

import { getIconCatalogPage, isIconCatalogCollection } from '../../utils/iconCatalog'

export default defineEventHandler((event) => {
  const collection = event.context.params?.collection
  if (!isIconCatalogCollection(collection)) {
    throw createError({
      statusCode: 404,
      statusMessage: 'Icon collection not found'
    })
  }

  return getIconCatalogPage(collection, getQuery(event))
})
