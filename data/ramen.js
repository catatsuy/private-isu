'use strict';

let jsdom = require('jsdom');
let moment = require('moment');
let co = require('co');
let fs = require('fs');
let https = require('https');

function loadTwilog(i) {
  return new Promise((resolve, reject) => {
    jsdom.env({
      url: 'http://twilog.org/catatsuy/search?word=pic.twitter.com&ao=a&page=' + i,
      done: (err, window) => {
        let $ = require('jquery')(window);
        let srcList = [];
        $('img.tl-image').each((i, img) => {
          let src = $(img).attr('src');
          //console.log(src);
          if (/jpg:small/.test(src)) {
            srcList.push(src.replace(/:small$/, ':large'));
          }
        });
        resolve(srcList);
      }
    });
  });
}

function loadImage(src) {
  return new Promise((resolve, reject) => {
    let filename = src.replace(/^.*\/([^\/]+\.jpg):large$/, '$1');
    //console.log(src, filename);

    https.get(src, (response) => {
      let file = fs.createWriteStream('img/' + filename);
      response.pipe(file);

      file.on('finish', () => {
        file.close(() => {
          resolve(filename);
        });
      });
      file.on('error', (err) => {
        console.log(err.message);
        reject(err.message);
      })
    }).on('error', (err) => {
      console.log(err.message);
      fs.unlink('img/' + filename);
      reject(err.message);
    });
  });
}

co(function *() {
  try {
    console.log('loading twilog...');

    for (let i = 0; i < 20; i++) {
      console.log(i);
      let srcList = yield loadTwilog(i);

      console.log('loading images...');
      for (let j = 0; j < srcList.length; j++) {
        console.log(srcList[j]);
        yield loadImage(srcList[j]);
      }
    }
  } catch (e) {
    console.log(e);
  }
});
