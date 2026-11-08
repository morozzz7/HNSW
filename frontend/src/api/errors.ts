export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly details?: string

  constructor(code: string, message: string, status = 0, details?: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.details = details
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError
}

const ERROR_MESSAGES: Record<string, string> = {
  INVALID_REQUEST: 'Некорректный запрос',
  DATASET_NOT_FOUND: 'Датасет не найден',
  INDEX_NOT_FOUND: 'Индекс ещё не построен',
  INDEX_NOT_READY: 'Индекс недоступен: идёт сборка или произошла ошибка',
  INVALID_DIMENSION: 'Размерность вектора не совпадает с индексом',
  K_EXCEEDS_N: 'k больше числа векторов в индексе',
  BUILD_FAILED: 'Не удалось построить индекс',
  SEARCH_FAILED: 'Ошибка поиска',
  FILE_TOO_LARGE: 'Файл больше 256 МиБ',
  UNSUPPORTED_FILE_TYPE: 'Допустимы только файлы .csv и .json',
  STORE_FULL: 'Хранилище датасетов заполнено',
  NOT_FOUND: 'Маршрут не найден',
  METHOD_NOT_ALLOWED: 'HTTP-метод не поддерживается',
  INTERNAL_ERROR: 'Внутренняя ошибка сервера',
  NETWORK_ERROR: 'Нет связи с сервером',
  TIMEOUT: 'Сервер не ответил вовремя',
  INVALID_RESPONSE: 'Сервер вернул неожиданный ответ',
}

export function errorMessage(error: unknown): string {
  if (isApiError(error)) {
    return ERROR_MESSAGES[error.code] ?? error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Неизвестная ошибка'
}

export function errorDetails(error: unknown): string | undefined {
  if (isApiError(error)) {
    if (ERROR_MESSAGES[error.code]) {
      return error.message
    }
    return error.details ?? error.message
  }
  return undefined
}

export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (failureCount >= 2) {
    return false
  }
  if (isApiError(error)) {
    return error.code === 'NETWORK_ERROR' || error.code === 'TIMEOUT' || error.status >= 500
  }
  return false
}
