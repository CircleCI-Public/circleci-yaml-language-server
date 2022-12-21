import {
  Diagnostic,
  Position,
  Range,
} from './types';

class DiagnosticList {
  readonly list: Diagnostic[];

  constructor(list: Diagnostic[]) {
    this.list = list;
  }

  includes(diagnostic: string | Partial<Diagnostic>) : boolean {
    if (typeof diagnostic === 'string') {
      return this.list.some((diag) => diag.message === diagnostic);
    }

    return this.list.some((diag) => areDiagnosticsEqual(diagnostic, diag));
  }

  sort() {
    this.list.sort(sortDiagnosticList);
  }
}

function areDiagnosticsEqual(a: Partial<Diagnostic>, b: Partial<Diagnostic>): boolean {
  if ('message' in a && 'message' in b && a.message !== b.message) {
    return false;
  }

  if ('severity' in a && 'severity' in b && a.severity !== b.severity) {
    return false;
  }

  if (a.range && b.range && areRangeEqual(a.range, b.range)) {
    return false;
  }

  if ('source' in a && 'source' in b && a.source !== b.source) {
    return false;
  }

  return true;
}

function areRangeEqual(a: Range, b: Range): boolean {
  return a.start.line === b.start.line
    && a.start.character === b.start.character
    && a.end.line === b.end.line
    && a.end.character === b.end.character;
}

function sortDiagnosticList(a: Diagnostic, b: Diagnostic) {
  return sortPosition(a.range.start, b.range.start)
  || sortPosition(a.range.end, b.range.end)
  || sortSeverity(a, b)
  || a.message.localeCompare(b.message);
}

function sortSeverity(diagA: Diagnostic, diagB: Diagnostic) {
  if (diagA.severity === diagB.severity) {
    return 0;
  }

  const a = diagA.severity ?? 99999;
  const b = diagB.severity ?? 99999;

  return a > b ? -1 : 1;
}

function sortPosition(a: Position, b: Position): number {
  if (a.line !== b.line) {
    return a.line > b.line ? -1 : 1;
  }

  if (a.character === b.character) {
    return 0;
  }

  return a.character > b.character ? -1 : 1;
}

export default DiagnosticList;
