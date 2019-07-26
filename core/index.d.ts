export as namespace core;

/**
 * @TJS-type integer
 * @minimum 0
 */
export type Counter = number;

/**
 * @TJS-type integer
 * @minimum 0
 */
export type Index = number;

/**
 * non-negative duration in nanoseconds
 * @TJS-type integer
 * @minimum 0
 */
export type NNDuration = number;

/**
 * @TJS-contentEncoding base64
 * @TJS-contentMediaType application/octet-stream
 */
export type Blob = string;
