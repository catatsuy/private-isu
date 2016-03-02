import 'babel-polyfill';
import Nightmare from 'nightmare';
import {expect} from 'chai';

const baseurl = process.env.TARGET_URL || "http://localhost:8080";

const option = {
    show: true,
      waitTimeout: 10000,
};

const shortWait = 100;

describe('e2etest', () => {
    let nightmare;

    before(async () => {
      nightmare = new Nightmare(option);
      await nightmare
      .goto(`${baseurl}/logout`)
      .goto(`${baseurl}/initialize`)
      .goto(`${baseurl}/`)
    });
    after(async () => {
      await nightmare.end();
    });

    beforeEach(async () => {
      await nightmare.wait(shortWait);
    });

    it('should register', async () => {
      await nightmare
      .goto(`${baseurl}/register`)
      .wait(shortWait)
      .type('input[name=account_name]', 'catatsuy')
      .type('input[name=password]', 'catatsuy')
      .click('input[type=submit]')
      .wait(shortWait)
      .goto(`${baseurl}/logout`)
      .wait(shortWait)
      .goto(`${baseurl}/login`)
      .type('input[name=account_name]', 'catatsuy')
      .type('input[name=password]', 'catatsuy')
      .click('input[type=submit]')
      .wait(shortWait)
      .goto(`${baseurl}/mypage`);

      const name = await nightmare.evaluate(function() {
        return document.querySelector('.isu-account-name a').textContent;
      });
      expect('catatsuyさん').to.equal(name);

});
