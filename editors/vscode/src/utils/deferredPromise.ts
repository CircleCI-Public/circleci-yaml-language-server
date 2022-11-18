export type DeferredPromise<T> = {
    promise: Promise<T>;
    resolve: (obj: T) => void;
    reject: (e: any) => void;
    created: number;
};

export function createDeferredPromise<T>(): DeferredPromise<T> {
    let resolve: (obj: T) => void;
    let reject: (e: any) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });
    let created = Date.now();
    //@ts-ignore
    return { resolve, reject, promise, created };
}
