class CustomEvent<T = unknown> extends Event {
  readonly detail: T | undefined;

  constructor(type: string, init: CustomEventInit<T>) {
    const { detail, ...initEvent } = init || {};

    super(type, initEvent);
    this.detail = detail;
  }
}

type CustomEventInit<T = unknown> = {
  detail?: T,
  cancelable?: boolean;
  composed?: boolean;
};

export default CustomEvent;
