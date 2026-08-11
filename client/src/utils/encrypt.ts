import CryptoJS from 'crypto-js'

/** 与 HotGo RequestEncryptKey / 管理端 aesEcb 保持一致 */
const defaultKey = 'f080a463654b2279'

export function encryptPassword(password: string, keyStr: string = defaultKey): string {
  const key = CryptoJS.enc.Utf8.parse(keyStr)
  const src = CryptoJS.enc.Utf8.parse(password)
  const encrypted = CryptoJS.AES.encrypt(src, key, {
    mode: CryptoJS.mode.ECB,
    padding: CryptoJS.pad.Pkcs7,
  })
  return encrypted.toString()
}
