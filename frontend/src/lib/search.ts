/**
 * 複数ワード AND 検索ユーティリティ
 *
 * 検索文字列をスペース（半角・全角）で分割し、
 * すべてのトークンがいずれかの対象フィールドに含まれていれば true を返す。
 *
 * - 大文字小文字を区別しない
 * - 空文字列の場合は常に true（フィルタなし）
 */

/**
 * 検索文字列をトークンに分割する。半角スペースと全角スペースの両方を区切り文字として扱う。
 */
export function tokenize(input: string): string[] {
  return input
    .trim()
    .toLowerCase()
    .split(/[\s　]+/)
    .filter(Boolean)
}

/**
 * すべてのトークンが、与えられたフィールドのいずれかに含まれているかを判定する（AND 検索）。
 *
 * @param searchInput - ユーザーが入力した検索文字列
 * @param fields - 検索対象フィールドの値の配列
 * @returns 検索文字列が空の場合は true、それ以外はすべてのトークンがいずれかのフィールドにマッチすれば true
 *
 * @example
 * ```ts
 * multiWordMatch('Omron リレー', ['ER2-C3', 'Omron制御リレー', 'Omron', 'リレー'])
 * // => true（'omron' は 'Omron制御リレー' or 'Omron' に含まれ、'リレー' も 'Omron制御リレー' or 'リレー' に含まれる）
 * ```
 */
export function multiWordMatch(searchInput: string, fields: string[]): boolean {
  const tokens = tokenize(searchInput)
  if (tokens.length === 0) return true
  const lowerFields = fields.map((f) => f.toLowerCase())
  return tokens.every((token) => lowerFields.some((field) => field.includes(token)))
}
