"use client";

import { clsx } from "clsx";
import { useMemo, useState, type ReactNode } from "react";
import { useAccount } from "wagmi";
import { CLI_INSTALL, CLI_VERSION, RELEASE_URL, SKILL_URL, skillInstall } from "@/config/agent";
import type { YellowNetwork } from "@/config/networks";
import { CodeBlock } from "@/components/ui/code-block";
import { useCopy } from "@/components/ui/copy-button";
import { IconCheck, IconCopy, IconLink, IconShield } from "@/components/ui/icons";
import { shortAddress } from "@/lib/format";
import { useNetwork } from "@/lib/network-context";

/** The CLI profile command for the network this site is showing. */
function networkCommand(n: YellowNetwork) {
  const gateway = n.gateway.replace(/\/+$/, "");
  return `yellow network add ${n.slug} --gateway ${gateway} --network-id ${n.networkId}`;
}

export function AgentView() {
  const { network } = useNetwork();
  const { address } = useAccount();
  const addNetwork = networkCommand(network);

  const setupPrompt = useMemo(
    () =>
      [
        "Set up Yellow Network for me:",
        `1. Install the Yellow command-line app v${CLI_VERSION} from ${RELEASE_URL} if the "yellow" command isn't available.`,
        `2. Install the Yellow Network skill from ${SKILL_URL} into your skills folder, keeping the whole yellow-network folder.`,
        `3. Add the ${network.name} network: ${addNetwork}. Show me its fingerprint so I can confirm it before you save it.`,
        `4. Create your own Yellow account with "yellow account create assistant --agent" and give me its address so I can fund it.`,
      ].join("\n"),
    [network.name, addNetwork],
  );

  return (
    <div className="pt-2 lg:pt-6">
      <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:items-start lg:gap-14">
        <header>
          <h1 className="type-h1">AI agent</h1>
          <p className="lede mt-3">
            Tell an AI assistant like Codex or Claude what you want in your own words. It checks balances, sends payments and follows them until they&apos;re
            final, from its own account with a budget you set.
          </p>
          <div className="mt-6 border-l-2 border-accent-edge pl-4">
            <h2 className="type-title text-[17px]">What is a skill?</h2>
            <p className="mt-1 max-w-[48ch] text-[15px] leading-relaxed text-on-canvas-muted">
              A small set of instructions you add to your assistant. The Yellow Network skill teaches it to use Yellow carefully, so you don&apos;t have to learn
              any commands.
            </p>
          </div>
        </header>
        <Conversation network={network.name} />
      </div>

      <section className="mt-12" aria-labelledby="setup-title">
        <h2 id="setup-title" className="type-title text-[24px]">
          Set it up
        </h2>
        <div className="mt-5 grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:items-start">
          <div className="sheet p-5 sm:p-7">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="type-title text-[19px]">Let your assistant do it</h3>
              <span className="type-label rounded-control bg-accent px-2 py-0.5 text-[12px] text-on-accent">Easiest</span>
            </div>
            <p className="mt-2 text-[15px] leading-relaxed text-muted">Copy this message into your AI assistant. It installs everything and tells you what to do next.</p>
            <CodeBlock className="mt-4" code={setupPrompt} label="setup message" prose />
            <p className="mt-4 text-[14px] leading-relaxed text-muted">
              Your assistant gets its own Yellow account. Send it only what it may spend, from Send in this wallet. Never paste recovery words or keys into a chat.
            </p>
          </div>
          <ManualSetup addNetwork={addNetwork} networkName={network.name} />
        </div>
      </section>

      <section className="mt-12" aria-labelledby="ask-title">
        <h2 id="ask-title" className="type-title text-[24px]">
          Try asking
        </h2>
        <ul className="mt-5 grid gap-2.5 sm:grid-cols-2">
          {[
            ...(address ? [`What's the Yellow balance of ${address}?`] : ["What's my Yellow balance?"]),
            "Send 50 USDC to 0x9a3F…9D10.",
            "Show my last five Yellow transfers.",
            "Send what's left of your budget back to me.",
          ].map((p) => (
            <PromptChip key={p} text={p} />
          ))}
        </ul>
        <p className="mt-3 text-[14px] text-on-canvas-muted">
          In Codex, start with <span className="font-mono text-[13px]">$yellow-network</span> to make sure it uses the skill.
        </p>
      </section>

      <section className="sheet mt-12 grid gap-6 p-5 sm:grid-cols-2 sm:p-7" aria-labelledby="control-title">
        <div>
          <span className="grid size-11 place-items-center rounded-control bg-accent text-on-accent">
            <IconShield size={22} />
          </span>
          <h2 id="control-title" className="type-title mt-4 text-[20px]">
            You set the budget
          </h2>
          <p className="mt-2 max-w-[42ch] text-[15px] leading-relaxed text-muted">
            Your assistant pays from its own Yellow account, so it can spend only what you move into it. Your main account stays out of its reach.
          </p>
        </div>
        <div className="grid gap-5 sm:grid-cols-2">
          <List title="Your assistant can" items={["Check balances", "Send payments you ask for", "Track a payment until it's final", "Never pay the same thing twice"]} />
          <List title="Only you can" items={["Fund its account", "Use your main account", "See your recovery words", "Take the budget back"]} />
        </div>
      </section>
    </div>
  );
}

function List({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <h3 className="type-label text-[15px]">{title}</h3>
      <ul className="mt-2 space-y-1.5">
        {items.map((i) => (
          <li key={i} className="flex items-center gap-2 text-[15px] text-muted">
            <IconCheck size={16} className="shrink-0 text-on-surface" />
            {i}
          </li>
        ))}
      </ul>
    </div>
  );
}

function PromptChip({ text }: { text: string }) {
  const { copied, copy } = useCopy();
  return (
    <li>
      <button
        type="button"
        onClick={() => copy(text)}
        className="group flex w-full items-center gap-3 rounded-field border border-line bg-surface p-4 text-left text-[15px] transition-colors hover:border-on-surface"
      >
        <span className="min-w-0 flex-1 break-words">“{text}”</span>
        <span className="shrink-0 text-muted group-hover:text-on-surface" aria-hidden>
          {copied ? <IconCheck size={17} /> : <IconCopy size={17} />}
        </span>
        <span className="sr-only">{copied ? "Copied" : "Copy"}</span>
      </button>
    </li>
  );
}

type AgentApp = "codex" | "claude" | "other";
const APPS: Array<{ id: AgentApp; label: string }> = [
  { id: "codex", label: "Codex" },
  { id: "claude", label: "Claude Code" },
  { id: "other", label: "Other" },
];

function ManualSetup({ addNetwork, networkName }: { addNetwork: string; networkName: string }) {
  const [app, setApp] = useState<AgentApp>("codex");
  return (
    <div className="sheet p-5 sm:p-7">
      <h3 className="type-title text-[19px]">Do it yourself</h3>
      <p className="mt-2 text-[15px] leading-relaxed text-muted">Run these in Terminal on macOS or Linux.</p>
      <ol className="mt-5 space-y-6">
        <Step n={1} title="Install Yellow">
          <CodeBlock code={CLI_INSTALL} label="install command" />
          <p className="mt-2 text-[13.5px] text-muted">
            Needs Go 1.25.6 or newer.{" "}
            <a href={RELEASE_URL} target="_blank" rel="noreferrer" className="underline underline-offset-2 hover:text-on-surface">
              Or download the app
            </a>{" "}
            for macOS, Windows or Linux.
          </p>
        </Step>
        <Step n={2} title="Add the skill to your assistant">
          <div role="tablist" aria-label="AI assistant" className="mb-3 grid grid-cols-3 rounded-field bg-surface-2 p-1">
            {APPS.map((a) => (
              <button
                key={a.id}
                role="tab"
                aria-selected={app === a.id}
                onClick={() => setApp(a.id)}
                className={clsx("type-label h-9 rounded-control text-[13.5px] transition-colors", app === a.id ? "bg-on-surface text-surface" : "text-muted")}
              >
                {a.label}
              </button>
            ))}
          </div>
          {app === "codex" && (
            <>
              <CodeBlock code={skillInstall("~/.agents/skills")} label="Codex skill install command" />
              <p className="mt-2 text-[13.5px] text-muted">Restart Codex if the skill doesn&apos;t show up.</p>
            </>
          )}
          {app === "claude" && (
            <>
              <CodeBlock code={skillInstall("~/.claude/skills")} label="Claude Code skill install command" />
              <p className="mt-2 text-[13.5px] text-muted">Start a new Claude Code session to pick it up.</p>
            </>
          )}
          {app === "other" && (
            <p className="text-[14.5px] leading-relaxed text-muted">
              Any assistant that supports skills (a folder with a SKILL.md file) can use it. Download the{" "}
              <span className="font-mono text-[13px] text-on-surface">yellow-network</span> folder and put all of it in your assistant&apos;s skills folder.
              <a
                href={SKILL_URL}
                target="_blank"
                rel="noreferrer"
                className="type-label mt-3 flex w-fit items-center gap-1.5 rounded-control border border-line px-3 py-2 text-[13.5px] text-on-surface hover:border-on-surface"
              >
                <IconLink size={15} /> Get the skill
              </a>
            </p>
          )}
        </Step>
        <Step n={3} title={`Add ${networkName} and your assistant's account`}>
          <CodeBlock code={`${addNetwork}\nyellow account create assistant --agent`} label="network and account commands" />
          <p className="mt-2 text-[13.5px] text-muted">Then send the new address the amount your assistant may spend.</p>
        </Step>
      </ol>
    </div>
  );
}

function Step({ n, title, children }: { n: number; title: string; children: ReactNode }) {
  return (
    <li className="grid grid-cols-[28px_minmax(0,1fr)] gap-x-3">
      <span className="type-label grid size-7 place-items-center rounded-control bg-on-surface text-[13px] text-surface" aria-hidden>
        {n}
      </span>
      <div className="min-w-0">
        <h4 className="type-label mb-2.5 pt-0.5 text-[16px]">{title}</h4>
        {children}
      </div>
    </li>
  );
}

/** An example exchange, so people can picture what using the skill is like. */
function Conversation({ network }: { network: string }) {
  return (
    <figure className="sheet p-4 sm:p-6">
      <figcaption className="mb-4 flex items-center justify-between text-[13px] text-muted">
        <span className="type-label text-on-surface">Example</span>
        <span>In your AI assistant</span>
      </figcaption>
      <div className="space-y-3 text-[15px] leading-relaxed">
        <Bubble from="you">What&apos;s in your budget?</Bubble>
        <Bubble from="agent">
          On {network}, my account <span className="font-mono text-[13px]">{shortAddress("0x996550701a55bcC2695c58ba16FB37d819B0A4dc")}</span> has:
          <dl className="num mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-0.5">
            {[
              ["200", "USDC"],
              ["25", "YELLOW"],
            ].map(([v, s]) => (
              <div key={s} className="contents">
                <dt className="type-label text-right">{v}</dt>
                <dd>{s}</dd>
              </div>
            ))}
          </dl>
        </Bubble>
        <Bubble from="you">Send 50 USDC to 0x9a3F…9D10</Bubble>
        <Bubble from="agent">Sending 50 USDC to 0x9a3F…9D10 from my Yellow account, with a fee cap of 1 YELLOW.</Bubble>
        <Sealed />
        <Bubble from="agent">Done. The payment is final. I have 150 USDC left in my budget.</Bubble>
      </div>
      <p className="mt-4 flex items-center gap-2 border-t border-line pt-4 text-[14px] text-muted">
        <IconShield size={17} className="shrink-0 text-on-surface" />
        It pays from its own account and can only spend what you put in it.
      </p>
    </figure>
  );
}

/** The network's confirmation, shown between the two assistant messages. */
function Sealed() {
  return (
    <div className="mx-auto flex w-fit items-center gap-2.5 rounded-field border border-line bg-surface px-3 py-2 text-[13.5px]">
      <span className="grid size-6 place-items-center rounded-control bg-accent text-on-accent">
        <IconCheck size={14} strokeWidth={3} />
      </span>
      <span>
        <span className="type-label">Sealed</span> <span className="text-muted">by Yellow Network</span>
      </span>
    </div>
  );
}

function Bubble({ from, children }: { from: "you" | "agent"; children: ReactNode }) {
  const you = from === "you";
  return (
    <div className={clsx("flex", you ? "justify-end" : "justify-start")}>
      <div className={clsx("max-w-[88%] rounded-field px-3.5 py-2.5", you ? "bg-on-surface text-surface" : "bg-surface-2 text-on-surface")}>
        <span className="sr-only">{you ? "You: " : "Assistant: "}</span>
        {children}
      </div>
    </div>
  );
}
