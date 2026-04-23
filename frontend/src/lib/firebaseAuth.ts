import { initializeApp, type FirebaseApp } from 'firebase/app'
import {
  browserLocalPersistence,
  createUserWithEmailAndPassword,
  getAuth,
  onIdTokenChanged,
  sendEmailVerification,
  setPersistence,
  signInWithEmailAndPassword,
  signOut,
  type Auth,
  type User,
} from 'firebase/auth'

import { setStoredToken } from './auth'
import { runtimeConfig } from './runtimeConfig'

let appInstance: FirebaseApp | null = null
let authInstance: Auth | null = null

export function isFirebaseAuthConfigured() {
  return !!runtimeConfig.firebaseApiKey && !!runtimeConfig.firebaseProjectId && !!runtimeConfig.firebaseAuthDomain
}

export function getFirebaseAuth() {
  if (!isFirebaseAuthConfigured()) {
    return null
  }
  if (!appInstance) {
    appInstance = initializeApp({
      apiKey: runtimeConfig.firebaseApiKey,
      authDomain: runtimeConfig.firebaseAuthDomain,
      projectId: runtimeConfig.firebaseProjectId,
      appId: runtimeConfig.firebaseAppId || undefined,
    })
  }
  if (!authInstance) {
    authInstance = getAuth(appInstance)
    void setPersistence(authInstance, browserLocalPersistence)
  }
  return authInstance
}

export function subscribeToIDTokenChange(callback: (user: User | null) => void) {
  const auth = getFirebaseAuth()
  if (!auth) {
    callback(null)
    return () => undefined
  }
  return onIdTokenChanged(auth, callback)
}

export async function signInWithIdentityPlatform(email: string, password: string) {
  const auth = getFirebaseAuth()
  if (!auth) {
    throw new Error('Firebase Auth runtime config is not set')
  }
  return signInWithEmailAndPassword(auth, email, password)
}

export async function signUpWithIdentityPlatform(email: string, password: string) {
  const auth = getFirebaseAuth()
  if (!auth) {
    throw new Error('Firebase Auth runtime config is not set')
  }
  return createUserWithEmailAndPassword(auth, email, password)
}

export async function syncStoredTokenFromIdentityPlatform(forceRefresh = false) {
  const auth = getFirebaseAuth()
  if (!auth?.currentUser) {
    throw new Error('No authenticated Identity Platform user')
  }
  const token = await auth.currentUser.getIdToken(forceRefresh)
  setStoredToken(token)
  return auth.currentUser
}

export function describeIdentityAuthError(caught: unknown, fallback: string) {
  const code = typeof caught === 'object' && caught !== null && 'code' in caught
    ? String((caught as { code?: unknown }).code ?? '')
    : ''

  switch (code) {
    case 'auth/email-already-in-use':
      return 'This email already exists in Identity Platform. Use Sign In instead. If it is already verified, you will be routed to Registration for this app.'
    case 'auth/invalid-credential':
    case 'auth/invalid-login-credentials':
      return 'Email or password is incorrect.'
    case 'auth/user-disabled':
      return 'This Identity Platform account is disabled.'
    case 'auth/too-many-requests':
      return 'Too many attempts. Wait a bit and try again.'
    case 'auth/weak-password':
      return 'Password is too weak.'
    default:
      return caught instanceof Error ? caught.message : fallback
  }
}

export async function sendVerificationEmail() {
  const auth = getFirebaseAuth()
  if (!auth?.currentUser) {
    throw new Error('No authenticated user for email verification')
  }
  await sendEmailVerification(auth.currentUser)
}

export async function refreshFirebaseUser() {
  const auth = getFirebaseAuth()
  if (!auth?.currentUser) {
    return null
  }
  await auth.currentUser.reload()
  return auth.currentUser
}

export async function signOutFromIdentityPlatform() {
  const auth = getFirebaseAuth()
  if (!auth) {
    return
  }
  await signOut(auth)
}
