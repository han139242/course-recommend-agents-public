function ok(data = {}, message = 'ok') {
  return {
    code: 0,
    message,
    data,
  }
}

function fail(message, code = 1, extra = {}) {
  return {
    code,
    message,
    ...extra,
  }
}

module.exports = {
  ok,
  fail,
}
