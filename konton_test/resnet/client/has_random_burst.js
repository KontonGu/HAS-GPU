import http from 'k6/http';
import encoding from 'k6/encoding';
import { group } from 'k6'
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { FormData } from 'https://jslib.k6.io/formdata/0.0.2/index.js';



export let options = {
  scenarios: {
    ramping: {
      executor: 'ramping-arrival-rate',
      startRate: 0,              // 起始 RPS
      timeUnit: '1s',            // RPS 的时间单位
      preAllocatedVUs: 50,       // 预分配的 VU 数量，确保足够承载请求
      maxVUs: 100,
      stages: [
        { target: 19, duration: '5s' },
        { target: 13, duration: '5s' },
        { target: 21, duration: '5s' },
        { target: 30, duration: '5s' },
        { target: 32, duration: '5s' },
        { target: 12, duration: '5s' },
        { target: 30, duration: '5s' },
        { target: 22, duration: '5s' },
        { target: 10, duration: '5s' },
        { target: 5, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 10, duration: '5s' },
        { target: 17, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 9, duration: '5s' },
        { target: 4, duration: '5s' },
        { target: 18, duration: '5s' },
        { target: 5, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 49, duration: '5s' },
        { target: 12, duration: '5s' },
        { target: 15, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 16, duration: '5s' },
        { target: 3, duration: '5s' },
        { target: 18, duration: '5s' },
        { target: 8, duration: '5s' },
        { target: 12, duration: '5s' },
        { target: 48, duration: '5s' },
        { target: 73, duration: '5s' },
        { target: 54, duration: '5s' },
        { target: 44, duration: '5s' },
        { target: 63, duration: '5s' },
        { target: 42, duration: '5s' },
        { target: 57, duration: '5s' },
        { target: 35, duration: '5s' },
        { target: 41, duration: '5s' },
        { target: 56, duration: '5s' },
        { target: 62, duration: '5s' },
        { target: 56, duration: '5s' },
        { target: 53, duration: '5s' },
        { target: 51, duration: '5s' },
        { target: 40, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 8, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 24, duration: '5s' },
        { target: 6, duration: '5s' },
        { target: 11, duration: '5s' },
        { target: 18, duration: '5s' },
        { target: 24, duration: '5s' },
        { target: 40, duration: '5s' },
        { target: 43, duration: '5s' },
        { target: 53, duration: '5s' },
        { target: 33, duration: '5s' },
        { target: 53, duration: '5s' },
        { target: 58, duration: '5s' },
        { target: 44, duration: '5s' },
        { target: 55, duration: '5s' },
        { target: 48, duration: '5s' },
        { target: 38, duration: '5s' },
        { target: 18, duration: '5s' },
        { target: 30, duration: '5s' },
        { target: 14, duration: '5s' },
        { target: 30, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 9, duration: '5s' },
        { target: 6, duration: '5s' },
        { target: 34, duration: '5s' },
        { target: 49, duration: '5s' },
        { target: 43, duration: '5s' },
        { target: 34, duration: '5s' },
        { target: 45, duration: '5s' },
        { target: 15, duration: '5s' },
        { target: 24, duration: '5s' },
        { target: 7, duration: '5s' },
        { target: 11, duration: '5s' },
        { target: 11, duration: '5s' },
        { target: 0, duration: '5s' },
        { target: 2, duration: '5s' },
        { target: 17, duration: '5s' },
        { target: 15, duration: '5s' },
        { target: 12, duration: '5s' }
    ],
    },
  },
};


const gateway = 'http://10.99.171.183:8080'
const image = open('car.jpg', 'b');
const fd = new FormData();
fd.append('payload', http.file(image, 'image.png', 'image/png'));
let resnet = {
        method: 'POST',
        // url: gateway + '/function/test-fastpod/predict',
        url: gateway + '/function/fastfunc-resnet/predict',
	// url: 'http://localhost:5000/predict',
        body: fd.body(), 
        params: {
            headers: {
	      'Content-Type': 'multipart/form-data; boundary=' + fd.boundary ,
	    },
        },
};
export default function () {
  const res = http.post(resnet.url, resnet.body, resnet.params)
  // const res = await http.asyncRequest("POST", resnet.url, resnet.body, resnet.params)
  check(res, {
    'is status 200': (r) => r.status === 200,
    //'check body': (r) => r.body.includes('car'),
  });
}