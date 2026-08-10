export const orderKeys={all:(siteId:string)=>['orders',siteId] as const,detail:(siteId:string,id:string)=>['orders',siteId,'detail',id] as const}
