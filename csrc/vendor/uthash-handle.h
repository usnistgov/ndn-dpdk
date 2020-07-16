// Moved out from uthash.h , see license in that file.

#ifndef NDNDPDK_VENDOR_UTHASH_HANDLE_H
#define NDNDPDK_VENDOR_UTHASH_HANDLE_H

#include <stdint.h>

typedef struct UT_hash_handle {
   struct UT_hash_table *tbl;
   void *prev;                       /* prev element in app order      */
   void *next;                       /* next element in app order      */
   struct UT_hash_handle *hh_prev;   /* previous hh in bucket order    */
   struct UT_hash_handle *hh_next;   /* next hh in bucket order        */
   void *key;                        /* ptr to enclosing struct's key  */
   unsigned keylen;                  /* enclosing struct's key len     */
   uint64_t hashv;                   /* result of hash-fcn(key)        */
} UT_hash_handle;

#endif // NDNDPDK_VENDOR_UTHASH_HANDLE_H
