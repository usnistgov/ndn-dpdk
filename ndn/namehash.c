#ifdef NAMEHASH_GENERATOR

// Makefile invokes this half of namehash.c to generate namehash.h

#include "../core/siphash.h"

int
main()
{
  uint8_t keybuf[SIPHASHKEY_SIZE];
  size_t n = fread(keybuf, 1, SIPHASHKEY_SIZE, stdin);
  if (n != SIPHASHKEY_SIZE) {
    return 3;
  }

  SipHashKey key;
  SipHashKey_FromBuffer(&key, keybuf);
  SipHash h;
  SipHash_Init(&h, &key);
  uint64_t emptyHash = SipHash_Final(&h);

  printf("#ifndef NDN_DPDK_NDN_NAMEHASH_H\n");
  printf("#define NDN_DPDK_NDN_NAMEHASH_H\n\n");

  printf("#include \"../core/siphash.h\"\n\n");

  printf("#define NAMEHASH_KEY \"");
  for (int i = 0; i < SIPHASHKEY_SIZE; ++i) {
    printf("\\x%02X", keybuf[i]);
  }
  printf("\"\n\n");

  printf("#define NAMEHASH_EMPTYHASH 0x%016" PRIX64 "\n\n", emptyHash);

  printf("extern SipHashKey theNameHashKey;\n\n");

  printf("#endif // NDN_DPDK_NDN_NAMEHASH_H\n");
}

#else // NAMEHASH_GENERATOR

#include "namehash.h"

SipHashKey theNameHashKey;

RTE_INIT(InitNameHashKey)
{
  SipHashKey_FromBuffer(&theNameHashKey, (const uint8_t*)NAMEHASH_KEY);

  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  assert(SipHash_Final(&h) == NAMEHASH_EMPTYHASH);
}

#endif // NAMEHASH_GENERATOR
