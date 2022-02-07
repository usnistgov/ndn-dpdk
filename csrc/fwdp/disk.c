#include "disk.h"

void
FwDisk_GotData(Packet* npkt, uintptr_t ctx)
{
  FwDisk* fwdisk = (FwDisk*)ctx;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  InputDemux_Dispatch(&fwdisk->output, npkt, &interest->name);
}
