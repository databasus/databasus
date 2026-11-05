export class RequestRevision {
  private revision = 0;

  GetCurrent(): number {
    return this.revision;
  }

  Invalidate(): number {
    return ++this.revision;
  }

  IsCurrent(revision: number): boolean {
    return this.revision === revision;
  }
}
