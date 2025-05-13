package controller

import "math"

type KalmanFilter struct {
	A, H, Q, R       float64
	x, P             float64
	Alpha, MinBuffer float64
}

func ImprovedKalmanFilter(initialEstimate, alpha, minBuffer float64) *KalmanFilter {
	return &KalmanFilter{
		A:         1,
		H:         1,
		Q:         1,
		R:         10,
		x:         initialEstimate,
		P:         1,
		Alpha:     alpha,
		MinBuffer: minBuffer,
	}
}

func (kf *KalmanFilter) Update(measurement float64) float64 {
	// Prediction
	xPred := kf.A * kf.x
	PPred := kf.A*kf.P*kf.A + kf.Q

	// update
	K := (PPred * kf.H) / (kf.H*PPred*kf.H + kf.R) // Kalman gain
	kf.x = xPred + K*(measurement-kf.H*xPred)
	kf.P = (1 - K*kf.H) * PPred

	// positive offset to ensure the prediction is larger than measurement
	kf.x += kf.Alpha * math.Sqrt(kf.P) // predicted req/s
	if kf.x < measurement+kf.MinBuffer {
		kf.x = measurement + kf.MinBuffer
	}

	return kf.x
}
