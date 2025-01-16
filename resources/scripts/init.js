Object.defineProperty(navigator, "webdriver", {
  configurable: true,
  enumerable: true,
  get: function () {
    return false;
  },
  set: undefined,
});
