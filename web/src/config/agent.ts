// Where the Yellow Network agent skill and CLI are published. Bump together
// with the CLI release (see cmd/yellow/README.md, "Scripts and agents").
export const CLI_VERSION = "0.9.0";
export const REPO = "layer-3/clearnet-sdk";

const tag = `v${CLI_VERSION}`;
const archiveRoot = `clearnet-sdk-${CLI_VERSION}`;

export const SKILL_URL = `https://github.com/${REPO}/tree/${tag}/skills/yellow-network`;
export const RELEASE_URL = `https://github.com/${REPO}/releases/tag/${tag}`;
export const CLI_INSTALL = `go install github.com/${REPO}/cmd/yellow@${tag}`;

/** Downloads only the skill folder from the release source archive into `dir`. */
export function skillInstall(dir: string) {
  return [
    `mkdir -p ${dir}`,
    `curl -fsSL https://github.com/${REPO}/archive/refs/tags/${tag}.tar.gz \\`,
    `  | tar -xz -C ${dir} --strip-components=2 ${archiveRoot}/skills/yellow-network`,
  ].join("\n");
}

export const INSTALL_PROMPT = `Install the Yellow Network agent skill from ${SKILL_URL} into your skills folder, keeping the whole yellow-network folder. Then check that the yellow command works by running "yellow version".`;
