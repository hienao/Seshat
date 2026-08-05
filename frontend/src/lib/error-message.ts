export function errorMessage(error: unknown, fallback = '操作失败，请稍后重试') {
  return error instanceof Error && error.message ? error.message : fallback
}
