/**
 * 删除 HTML 注释，反复迭代到不动点。
 *
 * 两个细节都是为了不让「删除」本身重新拼出注释边界
 * （js/incomplete-multi-character-sanitization 的形态）：
 *
 * 1. 迭代到不动点：单次替换会把 `<!<!-- -->-->` 这类嵌套构造重新拼成 `<!--`。
 * 2. 用空格而不是空串替换：空串删除会把注释两侧的字符粘在一起，既能拼出
 *    新的 `<!--`，也能把 `v-html`、`innerHTML` 这类被断言禁止的 token 拼出来。
 *    空格替换同时保住了 token 边界，语义上更接近「注释在这里，但不参与断言」。
 *
 * 与 `server/utils/iconSvg.ts` 的 `DROPPED_BLOCK_PATTERN` 处理保持同一 idiom。
 * 供「只断言真实代码、不被注释干扰」的静态契约测试使用。
 */
export function stripHtmlComments(value: string): string {
  let current = value
  for (;;) {
    const next = current.replace(/<!--[\s\S]*?-->/g, ' ')
    if (next === current) {
      return current
    }
    current = next
  }
}
