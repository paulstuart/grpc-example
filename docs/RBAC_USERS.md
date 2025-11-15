# RBAC Users

In order to properly demonstrate the power of this application we need to have a properly working RBAC system,
and such a system needs users to test. Let's finish the job:

1. Create an RBAC Engine that takes a data source for users, their roles, and the access patterns associated with the roles; the patterns would be mapped against the FullMethod attribute of the gRPC function that handles the given endpoint.
2. Users would leverage the User schema already created, extending it with roles that can match rpc endpoints and allow or deny them
3. Create a new working intercepter ClaimsApprover like FakeClaimsApprover in interceptors/jwt_auth.go line 42, except the real version would work from actual data:
   - It would check if the user_id in the Claims struct was allowed to access the rpc endpoint in question
   - It would use real data taken from the Storage interface of the server
   - It would create the data used by the Approver in a "batched" mechanism, that would use data at the time of creation to create it's own working dataset of who is allowed to access what resources. Ultimately that mechanism could be set to update every time the backing store is updated but for now at least loading it at startup and when explicitly triggered to refresh
4. To extend the comprehensive nature of this project and to give an easier way to update user data, let's create a ux/ package that is an HTMX server that provides a crud interface for the data accessed (via REST) in the API service.
5. This CRUD server could run as it's own app and be yet another container in the cluster to demonstrate full capability of this project.