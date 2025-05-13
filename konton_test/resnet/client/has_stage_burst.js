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
        // 每10秒增加10 RPS到100
        { target: 10, duration: '10s' },
        { target: 20, duration: '10s' },
        { target: 30, duration: '10s' },
        { target: 40, duration: '10s' },
        { target: 50, duration: '10s' },
        { target: 60, duration: '10s' },
        { target: 70, duration: '10s' },
        { target: 80, duration: '10s' },
        { target: 90, duration: '10s' },
        { target: 100, duration: '10s' },
        // 每10秒减少10 RPS到0
        { target: 90, duration: '10s' },
        { target: 80, duration: '10s' },
        { target: 70, duration: '10s' },
        { target: 60, duration: '10s' },
        { target: 50, duration: '10s' },
        { target: 40, duration: '10s' },
        { target: 30, duration: '10s' },
        { target: 20, duration: '10s' },
        { target: 10, duration: '10s' },
        { target: 0,  duration: '10s' },
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