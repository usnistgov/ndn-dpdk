import { gql, GqlClient } from "@usnistgov/ndn-dpdk";

export { gql };

export const client = new GqlClient("/graphql");
