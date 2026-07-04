export const sforumPreviewBanner = String.raw`   _____ ______
  / ___// ____/___  _______  ______ ___
  \__ \/ /_  / __ \/ ___/ / / / _  _  \
 ___/ / __/ / /_/ / /  / /_/ / / / / / /
/____/_/    \____/_/   \__,_/_/ /_/ /_/    SForum Web Preview
--------------------------------------------------`

export function printSForumPreviewBanner(output = console.log) {
  output(sforumPreviewBanner)
}
