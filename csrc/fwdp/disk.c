#include "disk.h"

void
FwDisk_GotData(Packet* npkt, uintptr_t ctx)
{
  FwDisk* fwdisk = (FwDisk*)ctx;
  bool accepted = InputDemux_Dispatch(&fwdisk->output, npkt);
  if (unlikely(!accepted)) {
    Packet_Free(npkt);
  }
}
