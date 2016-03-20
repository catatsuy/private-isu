import 'babel-polyfill';
import Nightmare from 'nightmare';
import {expect} from 'chai';

const baseurl = process.env.TARGET_URL || "http://localhost:8080";

const option = {
  show: true,
  waitTimeout: 10000,
};

const shortWait = 100;
const mediumWait = 2000;
const longWait = 5000;

describe('e2etest', () => {
  let nightmare;

  before(async () => {
    nightmare = new Nightmare(option);
    await nightmare
    .goto(`${baseurl}/initialize`)
    ;
  });
  after(async () => {
    await nightmare.end();
  });
  beforeEach(async () => {
    await nightmare
    .goto(`${baseurl}/logout`)
    await nightmare.wait(shortWait)
    ;
  });

  it('トップページが継ぎ足しできる', async () => {
    await nightmare
    .goto(`${baseurl}/`)
    .wait(shortWait)
    .scrollTo(100000, 0) // 一番下まで
    .wait(shortWait)
    .click('#isu-post-more-btn')
    .wait(longWait)
    ;

    const postLen1 = await nightmare.evaluate(() => {
      return document.querySelectorAll('.isu-post').length;
    });
    expect(39).to.equal(postLen1);

    await nightmare
    .scrollTo(100000, 0) // 一番下まで
    .wait(shortWait)
    .click('#isu-post-more-btn')
    .wait(longWait)
    ;

    const postLen2 = await nightmare.evaluate(() => {
      return document.querySelectorAll('.isu-post').length;
    });
    expect(58).to.equal(postLen2);
  });

  it('画像を投稿できる', async () => {
    await nightmare
    .goto(`${baseurl}/login`)
    .type('input[name=account_name]', 'mary')
    .type('input[name=password]', 'marymary')
    .click('input[type=submit]')
    .wait(longWait)
    .type('textarea[name=body]', 'あいうえお かきくけこ さしすせそ')
    .wait(shortWait)
    ;

    const urlAfterPost = await nightmare.evaluate(() => {
      const form = document.querySelector('.isu-submit form');
      const formData = new FormData(form);
      // 10x10の赤い正方形のgif
      const b64 = 'R0lGODdhBQAFAPAAAP8AAAAAACwAAAAABQAFAAACBISPqVgAOw==';

      // data URIをblobに変える方法: http://stackoverflow.com/a/11954337
      const binary = atob(b64);
      const array = [];
      for (let i = 0; i < binary.length; i++) {
        array.push(binary.charCodeAt(i));
      }
      const blob = new Blob([new Uint8Array(array)], {type: 'image/gif'});

      // formDataにblobを指定してXHRで送信することでファイルアップロードできる
      formData.append('file', blob, 'square.gif');

      // FormDataはXHRで送信するしかないので、送信後にリダイレクト先URLにlocation.hrefで遷移する
      const xhr = new XMLHttpRequest();
      xhr.open("POST", form.getAttribute('action'));
      xhr.onload = function() {
        location.href = xhr.responseURL;
      };
      xhr.send(formData);
    })
    .wait(mediumWait)
    .url()
    ;

    // 単体ページにリダイレクトされる
    expect(urlAfterPost).to.match(/\/posts\/(\d+)/);

    const id = RegExp.$1;

    // トップページにも反映される
    const postExists = await nightmare
    .goto(`${baseurl}/`)
    .wait(longWait)
    .exists('#pid_' + id)
    ;

    expect(postExists).to.be.true;
  });

  it('コメントできる', async () => {
    await nightmare
    .goto(`${baseurl}/login`)
    .type('input[name=account_name]', 'mary')
    .type('input[name=password]', 'marymary')
    .click('input[type=submit]')
    .wait(longWait)
    ;

    const [id, commentCount] = await nightmare.evaluate(() => {
      return [document.querySelector('.isu-post').id.replace('pid_', ''), document.querySelector('.isu-post-comment-count b').textContent];
    });

    const urlAfterComment = await nightmare
    .scrollTo(300, 0) // コメント欄が見えるまで
    .type('.isu-comment-form input[name=comment]', 'あいうえお かきくけこ')
    .click('.isu-comment-form input[type=submit]')
    .wait(mediumWait)
    .url()
    ;
    expect(urlAfterComment).to.equal(`${baseurl}/posts/${id}`);

    const commentCount2 = await nightmare.evaluate(() => {
      return document.querySelector('.isu-post-comment-count b').textContent;
    });
    expect(commentCount2).to.equal(+commentCount + 1 + '');
  });

  it('banできる', async () => {
    await nightmare
    .goto(`${baseurl}/login`)
    .type('input[name=account_name]', 'mary') // adminユーザー
    .type('input[name=password]', 'marymary')
    .click('input[type=submit]')
    .wait(shortWait)
    .goto(`${baseurl}/admin/banned`)
    .wait(shortWait)
    ;

    const name = await nightmare.evaluate(() => {
      return document.querySelector('input[name="uid[]"]').getAttribute('data-account-name');
    });

    const titleBeforeBan = await nightmare
    .goto(`${baseurl}/@${name}`)
    .title()
    ;
    expect(titleBeforeBan).to.equal('Iscogram'); // banされる前は普通に見れてる

    await nightmare
    .goto(`${baseurl}/admin/banned`)
    .wait(shortWait)
    .check('input[name="uid[]"]')
    .scrollTo(100000, 0) // 一番下まで
    .wait(shortWait)
    .click('input[type="submit"]')
    .wait(shortWait)
    ;

    const titleAfterBan = await nightmare
    .goto(`${baseurl}/@${name}`)
    .title()
    ;
    expect(titleAfterBan).to.equal('') // ステータスコードをチェックする方法が無いのでとりあえず
  });

  it('新規登録、ログアウト、ログインできる', async () => {
    const urlAfterRegister = await nightmare
    .goto(`${baseurl}/register`)
    .wait(mediumWait)
    .type('input[name=account_name]', 'catatsuy')
    .type('input[name=password]', 'catatsuy')
    .click('input[type=submit]')
    .wait(longWait)
    .url()
    ;
    expect(urlAfterRegister).to.equal(`${baseurl}/`);

    const name1 = await nightmare.evaluate(() => {
      return document.querySelector('.isu-account-name').textContent;
    });
    expect('catatsuy').to.equal(name1);

    const urlAfterLogin = await nightmare
    .goto(`${baseurl}/logout`)
    .wait(shortWait)
    .goto(`${baseurl}/login`)
    .type('input[name=account_name]', 'catatsuy')
    .type('input[name=password]', 'catatsuy')
    .click('input[type=submit]')
    .wait(longWait)
    .url()
    ;
    expect(urlAfterLogin).to.equal(`${baseurl}/`);

    const name2 = await nightmare.evaluate(() => {
      return document.querySelector('.isu-account-name').textContent;
    });
    expect('catatsuy').to.equal(name2);
  });
});
