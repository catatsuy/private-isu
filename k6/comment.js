import http from "k6/http";
import { check } from "k6";
import { parseHTML } from "k6/html";
import { url } from "./config.js";

export default function () {
  // ログイン
  const login_res = http.post(url("/login"), {
    account_name: "k6_user",
    password: "password",
  });
  check(login_res, {
    "is status 200": (r) => r.status === 200,
  });

  // ユーザーページへ移動
  const user_res = http.get(url("/@terra"));
  const user_doc = parseHTML(user_res.body);

  // csrf_token, post_id を取得
  const csrf_token = user_doc
    .find("input[name=csrf_token]")
    .first()
    .attr("value");
  const post_id = user_doc.find("input[name=post_id]").first().attr("value");

  // コメントを投稿する
  const comment_res = http.post(url("/comment"), {
    csrf_token: csrf_token,
    post_id: post_id,
    comment: "Hello k6!",
  });
  check(comment_res, {
    "is status 200": (r) => r.status === 200,
  });
}
