"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isAppleSilicon = void 0;
const os = require("os");
/**
 * Check if the user is running on an Apple Silicon chipset
 * @returns true if the processor is an Apple one (such as the M1)
 */
function isAppleSilicon() {
    return os.cpus().some((cpu) => cpu.model.startsWith('Apple '));
}
exports.isAppleSilicon = isAppleSilicon;
//# sourceMappingURL=appleSilicon.js.map