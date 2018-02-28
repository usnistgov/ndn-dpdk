#include "rxburst.h"

FaceRxBurst*
FaceRxBurst_New(uint16_t capacity)
{
  size_t size = sizeof(FaceRxBurst) + 3 * capacity * sizeof(Packet*);
  FaceRxBurst* burst = rte_malloc("FaceRxBurst", size, 0);
  burst->capacity = capacity;
  return burst;
}

void
FaceRxBurst_Close(FaceRxBurst* burst)
{
  return rte_free(burst);
}
