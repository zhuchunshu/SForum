export type CommentPresentationMode = 'tree' | 'flat'

export type CommentBranchNode = {
  children?: readonly CommentBranchNode[]
}

export type CommentBranchPresentation = {
  connectionRail: boolean
  collapsible: boolean
  followUpCount: number
  indentation: 0 | 1
}

export function countCommentDescendants(children: readonly CommentBranchNode[] = []): number {
  return children.reduce(
    (total, child) => total + 1 + countCommentDescendants(child.children),
    0
  )
}

export function commentBranchPresentation(
  mode: CommentPresentationMode,
  depth: number,
  children: readonly CommentBranchNode[] = [],
  collapseFromDepth = 2
): CommentBranchPresentation {
  const normalizedDepth = Math.max(0, Math.trunc(depth))
  const normalizedCollapseDepth = Math.max(1, Math.trunc(collapseFromDepth))
  const hasChildren = children.length > 0
  const isTree = mode === 'tree'

  return {
    // 根分支拥有唯一连接线，后续层级沿用同一视觉缩进。
    connectionRail: isTree && normalizedDepth === 0 && hasChildren,
    collapsible: isTree && hasChildren && normalizedDepth + 1 === normalizedCollapseDepth,
    followUpCount: countCommentDescendants(children),
    indentation: isTree && normalizedDepth > 0 ? 1 : 0
  }
}
