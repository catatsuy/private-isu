import initialize from "./initialize.js";
import comment from "./comment.js";
import postimage from "./postimage.js";

// k6 が各関数を実行できるように
export { initialize, comment, postimage };

// 統合シナリオの定義
export const options = {
  scenarios: {
    // 初期化シナリオは1回のみ1並列で実行
    initialize: {
      executor: "shared-iterations",
      vus: 1,
      iterations: 1,
      maxDuration: "10s",
      exec: "initialize",
    },
    // コメントシナリオは4並列で実行30秒実行
    comment: {
      executor: "constant-vus",
      vus: 4,
      duration: "30s",
      exec: "comment",
      startTime: "12s", // initialize が終わってから
    },
    // 画像投稿シナリオは2並列で実行30秒実行
    postimage: {
      executor: "constant-vus",
      vus: 2,
      duration: "30s",
      exec: "postimage",
      startTime: "12s", // initialize が終わってから
    },
  },
};

// default export は空でOK
export default function () {}
