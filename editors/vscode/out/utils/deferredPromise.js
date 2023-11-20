"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createDeferredPromise = void 0;
function createDeferredPromise() {
    let resolve;
    let reject;
    const promise = new Promise((res, rej) => {
        resolve = res;
        reject = rej;
    });
    let created = Date.now();
    //@ts-ignore
    return { resolve, reject, promise, created };
}
exports.createDeferredPromise = createDeferredPromise;
//# sourceMappingURL=deferredPromise.js.map