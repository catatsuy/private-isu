import 'express-session';

declare module 'express-session' {
  interface SessionData {
    /** ログインユーザー ID */
    userId?: number;
    /** CSRF トークン */
    csrfToken?: string;
  }
}
