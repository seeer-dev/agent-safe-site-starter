import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { adminTransport } from '@sitecore/admin-transport'
import { orderKeys } from './query-keys'

export function useCancelOrder(siteId:string,id:string){
  const client=useQueryClient()
  return useMutation({
    mutationFn:({expectedRevision,commandId}:{expectedRevision:number;commandId:string})=>adminTransport.cancelOrder({siteId,id,expectedRevision,commandId}),
    retry:false,
    onSuccess:()=>client.invalidateQueries({queryKey:orderKeys.detail(siteId,id),exact:true}),
  })
}
