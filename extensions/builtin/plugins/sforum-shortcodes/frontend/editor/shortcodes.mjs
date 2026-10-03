export const apiVersion = 1;

const referenceKinds = Object.freeze([
  "user",
  "topic",
  "comment",
  "category",
  "friend-links",
]);
const protectedKinds = Object.freeze(["login", "reply", "only-author"]);

export function createExtensions() {
  return [];
}

export function createCommands() {
  return {
    async openReferenceMenu(context) {
      if (context.disabled) return false;
      const resources = context.resourceKind === "comment"
        ? [...referenceKinds, ...protectedKinds]
        : [...referenceKinds, "login", "reply"];
      const result = await context.host.openReferenceDialog(resources);
      if (result.action === "upsert") {
        if (protectedKinds.includes(result.kind)) {
          return context.host.upsertProtected(result.kind);
        }
        return context.host.upsertReference(result.kind, result.id);
      }
      if (result.action === "delete") {
        return context.host.deleteReference();
      }
      context.host.focusEditor();
      return false;
    },
  };
}
