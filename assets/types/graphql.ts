import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Any: { input: unknown; output: unknown; }
  Int64: { input: number; output: number; }
  Map: { input: Record<string, string>; output: Record<string, string>; }
  Time: { input: string; output: string; }
};

export type Container = {
  __typename?: 'Container';
  command: Scalars['String']['output'];
  created: Scalars['Time']['output'];
  group?: Maybe<Scalars['String']['output']>;
  health?: Maybe<Scalars['String']['output']>;
  host?: Maybe<Scalars['String']['output']>;
  id: Scalars['String']['output'];
  image: Scalars['String']['output'];
  labels?: Maybe<Scalars['Map']['output']>;
  name: Scalars['String']['output'];
  startedAt: Scalars['Time']['output'];
  state: Scalars['String']['output'];
};

export type Dispatcher = {
  __typename?: 'Dispatcher';
  id: Scalars['Int']['output'];
  name: Scalars['String']['output'];
  template?: Maybe<Scalars['String']['output']>;
  type: Scalars['String']['output'];
  url?: Maybe<Scalars['String']['output']>;
};

export type DispatcherInput = {
  name: Scalars['String']['input'];
  template?: InputMaybe<Scalars['String']['input']>;
  type: Scalars['String']['input'];
  url?: InputMaybe<Scalars['String']['input']>;
};

export type LogEvent = {
  __typename?: 'LogEvent';
  containerId?: Maybe<Scalars['String']['output']>;
  id: Scalars['Int']['output'];
  level?: Maybe<Scalars['String']['output']>;
  message?: Maybe<Scalars['Any']['output']>;
  stream?: Maybe<Scalars['String']['output']>;
  timestamp: Scalars['Int64']['output'];
  type?: Maybe<Scalars['String']['output']>;
};

export type Mutation = {
  __typename?: 'Mutation';
  createDispatcher: Dispatcher;
  createNotificationRule: NotificationRule;
  deleteDispatcher: Scalars['Boolean']['output'];
  deleteNotificationRule: Scalars['Boolean']['output'];
  previewExpression: PreviewResult;
  replaceNotificationRule: NotificationRule;
  updateDispatcher: Dispatcher;
  updateNotificationRule: NotificationRule;
};


export type MutationCreateDispatcherArgs = {
  input: DispatcherInput;
};


export type MutationCreateNotificationRuleArgs = {
  input: NotificationRuleInput;
};


export type MutationDeleteDispatcherArgs = {
  id: Scalars['Int']['input'];
};


export type MutationDeleteNotificationRuleArgs = {
  id: Scalars['Int']['input'];
};


export type MutationPreviewExpressionArgs = {
  input: PreviewInput;
};


export type MutationReplaceNotificationRuleArgs = {
  id: Scalars['Int']['input'];
  input: NotificationRuleInput;
};


export type MutationUpdateDispatcherArgs = {
  id: Scalars['Int']['input'];
  input: DispatcherInput;
};


export type MutationUpdateNotificationRuleArgs = {
  id: Scalars['Int']['input'];
  input: NotificationRuleUpdateInput;
};

export type NotificationRule = {
  __typename?: 'NotificationRule';
  containerExpression: Scalars['String']['output'];
  dispatcher?: Maybe<Dispatcher>;
  enabled: Scalars['Boolean']['output'];
  id: Scalars['Int']['output'];
  lastTriggeredAt?: Maybe<Scalars['Time']['output']>;
  logExpression: Scalars['String']['output'];
  name: Scalars['String']['output'];
  triggerCount: Scalars['Int64']['output'];
  triggeredContainers: Scalars['Int']['output'];
};

export type NotificationRuleInput = {
  containerExpression: Scalars['String']['input'];
  dispatcherId: Scalars['Int']['input'];
  enabled: Scalars['Boolean']['input'];
  logExpression: Scalars['String']['input'];
  name: Scalars['String']['input'];
};

export type NotificationRuleUpdateInput = {
  containerExpression?: InputMaybe<Scalars['String']['input']>;
  dispatcherId?: InputMaybe<Scalars['Int']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  logExpression?: InputMaybe<Scalars['String']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
};

export type PreviewInput = {
  containerExpression: Scalars['String']['input'];
  logExpression?: InputMaybe<Scalars['String']['input']>;
};

export type PreviewResult = {
  __typename?: 'PreviewResult';
  containerError?: Maybe<Scalars['String']['output']>;
  logError?: Maybe<Scalars['String']['output']>;
  matchedContainers: Array<Container>;
  matchedLogs: Array<LogEvent>;
  totalLogs: Scalars['Int']['output'];
};

export type Query = {
  __typename?: 'Query';
  dispatcher?: Maybe<Dispatcher>;
  dispatchers: Array<Dispatcher>;
  notificationRule?: Maybe<NotificationRule>;
  notificationRules: Array<NotificationRule>;
  releases: Array<Release>;
};


export type QueryDispatcherArgs = {
  id: Scalars['Int']['input'];
};


export type QueryNotificationRuleArgs = {
  id: Scalars['Int']['input'];
};

export type Release = {
  __typename?: 'Release';
  body: Scalars['String']['output'];
  breaking: Scalars['Int']['output'];
  bugFixes: Scalars['Int']['output'];
  createdAt: Scalars['Time']['output'];
  features: Scalars['Int']['output'];
  htmlUrl: Scalars['String']['output'];
  latest: Scalars['Boolean']['output'];
  mentionsCount: Scalars['Int']['output'];
  name: Scalars['String']['output'];
  tag: Scalars['String']['output'];
};

export type GetNotificationRulesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetNotificationRulesQuery = { __typename?: 'Query', notificationRules: Array<{ __typename?: 'NotificationRule', id: number, name: string, enabled: boolean, containerExpression: string, logExpression: string, triggerCount: number, triggeredContainers: number, lastTriggeredAt?: string | null, dispatcher?: { __typename?: 'Dispatcher', id: number, name: string, type: string, url?: string | null } | null }> };

export type GetDispatchersQueryVariables = Exact<{ [key: string]: never; }>;


export type GetDispatchersQuery = { __typename?: 'Query', dispatchers: Array<{ __typename?: 'Dispatcher', id: number, name: string, type: string, url?: string | null, template?: string | null }> };

export type CreateNotificationRuleMutationVariables = Exact<{
  input: NotificationRuleInput;
}>;


export type CreateNotificationRuleMutation = { __typename?: 'Mutation', createNotificationRule: { __typename?: 'NotificationRule', id: number, name: string, enabled: boolean, containerExpression: string, logExpression: string, dispatcher?: { __typename?: 'Dispatcher', id: number, name: string, type: string } | null } };

export type UpdateNotificationRuleMutationVariables = Exact<{
  id: Scalars['Int']['input'];
  input: NotificationRuleUpdateInput;
}>;


export type UpdateNotificationRuleMutation = { __typename?: 'Mutation', updateNotificationRule: { __typename?: 'NotificationRule', id: number, name: string, enabled: boolean, containerExpression: string, logExpression: string, dispatcher?: { __typename?: 'Dispatcher', id: number, name: string, type: string } | null } };

export type ReplaceNotificationRuleMutationVariables = Exact<{
  id: Scalars['Int']['input'];
  input: NotificationRuleInput;
}>;


export type ReplaceNotificationRuleMutation = { __typename?: 'Mutation', replaceNotificationRule: { __typename?: 'NotificationRule', id: number, name: string, enabled: boolean, containerExpression: string, logExpression: string, dispatcher?: { __typename?: 'Dispatcher', id: number, name: string, type: string } | null } };

export type DeleteNotificationRuleMutationVariables = Exact<{
  id: Scalars['Int']['input'];
}>;


export type DeleteNotificationRuleMutation = { __typename?: 'Mutation', deleteNotificationRule: boolean };

export type CreateDispatcherMutationVariables = Exact<{
  input: DispatcherInput;
}>;


export type CreateDispatcherMutation = { __typename?: 'Mutation', createDispatcher: { __typename?: 'Dispatcher', id: number, name: string, type: string, url?: string | null } };

export type UpdateDispatcherMutationVariables = Exact<{
  id: Scalars['Int']['input'];
  input: DispatcherInput;
}>;


export type UpdateDispatcherMutation = { __typename?: 'Mutation', updateDispatcher: { __typename?: 'Dispatcher', id: number, name: string, type: string, url?: string | null, template?: string | null } };

export type DeleteDispatcherMutationVariables = Exact<{
  id: Scalars['Int']['input'];
}>;


export type DeleteDispatcherMutation = { __typename?: 'Mutation', deleteDispatcher: boolean };

export type PreviewExpressionMutationVariables = Exact<{
  input: PreviewInput;
}>;


export type PreviewExpressionMutation = { __typename?: 'Mutation', previewExpression: { __typename?: 'PreviewResult', containerError?: string | null, logError?: string | null, totalLogs: number, matchedContainers: Array<{ __typename?: 'Container', id: string, name: string, image: string, host?: string | null }>, matchedLogs: Array<{ __typename?: 'LogEvent', id: number, type?: string | null, message?: unknown | null, timestamp: number, level?: string | null, stream?: string | null }> } };

export type GetReleasesQueryVariables = Exact<{ [key: string]: never; }>;


export type GetReleasesQuery = { __typename?: 'Query', releases: Array<{ __typename?: 'Release', name: string, mentionsCount: number, tag: string, body: string, createdAt: string, htmlUrl: string, latest: boolean, features: number, bugFixes: number, breaking: number }> };


export const GetNotificationRulesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetNotificationRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"notificationRules"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"containerExpression"}},{"kind":"Field","name":{"kind":"Name","value":"logExpression"}},{"kind":"Field","name":{"kind":"Name","value":"triggerCount"}},{"kind":"Field","name":{"kind":"Name","value":"triggeredContainers"}},{"kind":"Field","name":{"kind":"Name","value":"lastTriggeredAt"}},{"kind":"Field","name":{"kind":"Name","value":"dispatcher"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]}}]} as unknown as DocumentNode<GetNotificationRulesQuery, GetNotificationRulesQueryVariables>;
export const GetDispatchersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetDispatchers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dispatchers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"template"}}]}}]}}]} as unknown as DocumentNode<GetDispatchersQuery, GetDispatchersQueryVariables>;
export const CreateNotificationRuleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateNotificationRule"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NotificationRuleInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createNotificationRule"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"containerExpression"}},{"kind":"Field","name":{"kind":"Name","value":"logExpression"}},{"kind":"Field","name":{"kind":"Name","value":"dispatcher"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}}]}}]}}]} as unknown as DocumentNode<CreateNotificationRuleMutation, CreateNotificationRuleMutationVariables>;
export const UpdateNotificationRuleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateNotificationRule"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NotificationRuleUpdateInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateNotificationRule"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"containerExpression"}},{"kind":"Field","name":{"kind":"Name","value":"logExpression"}},{"kind":"Field","name":{"kind":"Name","value":"dispatcher"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateNotificationRuleMutation, UpdateNotificationRuleMutationVariables>;
export const ReplaceNotificationRuleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReplaceNotificationRule"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"NotificationRuleInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"replaceNotificationRule"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"containerExpression"}},{"kind":"Field","name":{"kind":"Name","value":"logExpression"}},{"kind":"Field","name":{"kind":"Name","value":"dispatcher"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}}]}}]}}]}}]} as unknown as DocumentNode<ReplaceNotificationRuleMutation, ReplaceNotificationRuleMutationVariables>;
export const DeleteNotificationRuleDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteNotificationRule"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteNotificationRule"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<DeleteNotificationRuleMutation, DeleteNotificationRuleMutationVariables>;
export const CreateDispatcherDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateDispatcher"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DispatcherInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createDispatcher"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}}]}}]}}]} as unknown as DocumentNode<CreateDispatcherMutation, CreateDispatcherMutationVariables>;
export const UpdateDispatcherDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateDispatcher"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"DispatcherInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateDispatcher"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}},{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"url"}},{"kind":"Field","name":{"kind":"Name","value":"template"}}]}}]}}]} as unknown as DocumentNode<UpdateDispatcherMutation, UpdateDispatcherMutationVariables>;
export const DeleteDispatcherDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"DeleteDispatcher"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"id"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"Int"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"deleteDispatcher"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"id"},"value":{"kind":"Variable","name":{"kind":"Name","value":"id"}}}]}]}}]} as unknown as DocumentNode<DeleteDispatcherMutation, DeleteDispatcherMutationVariables>;
export const PreviewExpressionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"PreviewExpression"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"PreviewInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"previewExpression"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"containerError"}},{"kind":"Field","name":{"kind":"Name","value":"logError"}},{"kind":"Field","name":{"kind":"Name","value":"matchedContainers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"image"}},{"kind":"Field","name":{"kind":"Name","value":"host"}}]}},{"kind":"Field","name":{"kind":"Name","value":"matchedLogs"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"timestamp"}},{"kind":"Field","name":{"kind":"Name","value":"level"}},{"kind":"Field","name":{"kind":"Name","value":"stream"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalLogs"}}]}}]}}]} as unknown as DocumentNode<PreviewExpressionMutation, PreviewExpressionMutationVariables>;
export const GetReleasesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetReleases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"releases"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"mentionsCount"}},{"kind":"Field","name":{"kind":"Name","value":"tag"}},{"kind":"Field","name":{"kind":"Name","value":"body"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}},{"kind":"Field","name":{"kind":"Name","value":"htmlUrl"}},{"kind":"Field","name":{"kind":"Name","value":"latest"}},{"kind":"Field","name":{"kind":"Name","value":"features"}},{"kind":"Field","name":{"kind":"Name","value":"bugFixes"}},{"kind":"Field","name":{"kind":"Name","value":"breaking"}}]}}]}}]} as unknown as DocumentNode<GetReleasesQuery, GetReleasesQueryVariables>;