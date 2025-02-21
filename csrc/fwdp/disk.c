#include "disk.h"

void
FwDisk_GotData(Packet* npkt, uintptr_t ctx) {
  FwDisk* fwdisk = (FwDisk*)ctx;
  uint64_t rejectMask = InputDemux_Dispatch(&fwdisk->output, &npkt, 1);
  if (unlikely(rejectMask != 0)) {
    Packet_Free(npkt);
  }
}
