import * as os from 'os';

/**
 * Check if the user is running on an Apple Silicon chipset
 * @returns true if the processor is an Apple one (such as the M1)
 */
export function isAppleSilicon(): boolean {
    return os.cpus().some((cpu) => cpu.model.startsWith('Apple '));
}
