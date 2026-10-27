import type { Metadata } from "next";
import type { Locale } from "@/app/i18n";
import EsContent, { metadata as esMetadata } from "./content/es";
import FrContent, { metadata as frMetadata } from "./content/fr";
import IdContent, { metadata as idMetadata } from "./content/id";
import PtContent, { metadata as ptMetadata } from "./content/pt";
import RuContent, { metadata as ruMetadata } from "./content/ru";
import ZhContent, { metadata as zhMetadata } from "./content/zh";

const CONTENT_BY_LOCALE: Record<Locale, { metadata: Metadata; Content: () => React.ReactNode }> = {
  ru: { metadata: ruMetadata, Content: RuContent },
  es: { metadata: esMetadata, Content: EsContent },
  pt: { metadata: ptMetadata, Content: PtContent },
  zh: { metadata: zhMetadata, Content: ZhContent },
  fr: { metadata: frMetadata, Content: FrContent },
  id: { metadata: idMetadata, Content: IdContent },
};

export async function generateMetadata({
  params,
}: {
  params: Promise<{ lang: string }>;
}): Promise<Metadata> {
  const { lang } = await params;
  return CONTENT_BY_LOCALE[lang as Locale].metadata;
}

export default async function LocalizedPage({ params }: { params: Promise<{ lang: string }> }) {
  const { lang } = await params;
  const { Content } = CONTENT_BY_LOCALE[lang as Locale];
  return <Content />;
}
