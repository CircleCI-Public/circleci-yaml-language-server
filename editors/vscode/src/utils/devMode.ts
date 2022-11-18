export function isInDevMode(): boolean {
    return process.env.CCI_DEV === 'true';
}
