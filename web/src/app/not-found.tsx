import { ButtonLink } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="pt-6 lg:pt-12">
      <h1 className="type-h1">Not here</h1>
      <p className="lede mt-3">This page doesn&apos;t exist. Your account is one tap away.</p>
      <ButtonLink href="/" size="lg" className="mt-8">
        Go to account
      </ButtonLink>
    </div>
  );
}
