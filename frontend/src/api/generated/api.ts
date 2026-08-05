/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface HandlerSetUserRoleRequest {
  is_admin?: boolean;
}

export interface ResponseResponse {
  code?: number;
  data?: any;
  message?: string;
}

export interface ServiceChangePasswordRequest {
  /** @minLength 6 */
  new_password: string;
  old_password: string;
}

export interface ServiceLoginRequest {
  password: string;
  username: string;
}

export interface ServiceRegisterRequest {
  /** @minLength 6 */
  password: string;
  /**
   * @minLength 3
   * @maxLength 50
   */
  username: string;
}

export interface ServiceSystemSettingsResponse {
  allow_register?: boolean;
}

export interface ServiceTokenResponse {
  expires_at?: number;
  token?: string;
}

export interface ServiceUpdateSystemSettingsRequest {
  allow_register?: boolean;
}

export interface ServiceUserResponse {
  created_at?: string;
  id?: number;
  is_admin?: boolean;
  username?: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseFormat;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<RequestParams | void> | RequestParams | void;
  customFetch?: typeof fetch;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown>
  extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "//localhost:8080";
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private abortControllers = new Map<CancelToken, AbortController>();
  private customFetch = (...fetchParams: Parameters<typeof fetch>) =>
    fetch(...fetchParams);

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected encodeQueryParam(key: string, value: any) {
    const encodedKey = encodeURIComponent(key);
    return `${encodedKey}=${encodeURIComponent(typeof value === "number" ? value : `${value}`)}`;
  }

  protected addQueryParam(query: QueryParamsType, key: string) {
    return this.encodeQueryParam(key, query[key]);
  }

  protected addArrayQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];
    return value.map((v: any) => this.encodeQueryParam(key, v)).join("&");
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter(
      (key) => "undefined" !== typeof query[key],
    );
    return keys
      .map((key) =>
        Array.isArray(query[key])
          ? this.addArrayQueryParam(query, key)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.JsonApi]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.Text]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    [ContentType.FormData]: (input: any) => {
      if (input instanceof FormData) {
        return input;
      }

      return Object.keys(input || {}).reduce((formData, key) => {
        const property = input[key];
        formData.append(
          key,
          property instanceof Blob
            ? property
            : typeof property === "object" && property !== null
              ? JSON.stringify(property)
              : `${property}`,
        );
        return formData;
      }, new FormData());
    },
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
  };

  protected mergeRequestParams(
    params1: RequestParams,
    params2?: RequestParams,
  ): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected createAbortSignal = (
    cancelToken: CancelToken,
  ): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = async <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format,
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.baseApiParams.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== ContentType.FormData
            ? { "Content-Type": type }
            : {}),
        },
        signal:
          (cancelToken
            ? this.createAbortSignal(cancelToken)
            : requestParams.signal) || null,
        body:
          typeof body === "undefined" || body === null
            ? null
            : payloadFormatter(body),
      },
    ).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = null as unknown as T;
      r.error = null as unknown as E;

      const responseToParse = responseFormat ? response.clone() : response;
      const data = !responseFormat
        ? r
        : await responseToParse[responseFormat]()
            .then((data) => {
              if (r.ok) {
                r.data = data;
              } else {
                r.error = data;
              }
              return r;
            })
            .catch((e) => {
              r.error = e;
              return r;
            });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title BaseGoApp API
 * @version 1.0
 * @baseUrl //localhost:8080
 * @contact
 *
 * BaseGoApp 模板工程 API 文档
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  api = {
    /**
     * @description 获取所有用户列表（需要管理员权限）
     *
     * @tags 管理
     * @name AdminUsersList
     * @summary 获取用户列表
     * @request GET:/api/admin/users
     * @secure
     */
    adminUsersList: (params: RequestParams = {}) =>
      this.request<
        ResponseResponse & {
          data?: ServiceUserResponse[];
        },
        ResponseResponse
      >({
        path: `/api/admin/users`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description 将用户提升或降级管理员权限（需要管理员权限）
     *
     * @tags 管理
     * @name AdminUsersRoleUpdate
     * @summary 设置用户角色
     * @request PUT:/api/admin/users/{id}/role
     * @secure
     */
    adminUsersRoleUpdate: (
      id: number,
      request: HandlerSetUserRoleRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseResponse, ResponseResponse>({
        path: `/api/admin/users/${id}/role`,
        method: "PUT",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 用户登录获取 JWT Token
     *
     * @tags 认证
     * @name AuthLoginCreate
     * @summary 用户登录
     * @request POST:/api/auth/login
     */
    authLoginCreate: (
      request: ServiceLoginRequest,
      params: RequestParams = {},
    ) =>
      this.request<
        ResponseResponse & {
          data?: ServiceTokenResponse;
        },
        ResponseResponse
      >({
        path: `/api/auth/login`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 用户退出登录（前端清除 Token）
     *
     * @tags 认证
     * @name AuthLogoutCreate
     * @summary 用户退出
     * @request POST:/api/auth/logout
     * @secure
     */
    authLogoutCreate: (params: RequestParams = {}) =>
      this.request<ResponseResponse, any>({
        path: `/api/auth/logout`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description 创建新用户账户
     *
     * @tags 认证
     * @name AuthRegisterCreate
     * @summary 用户注册
     * @request POST:/api/auth/register
     */
    authRegisterCreate: (
      request: ServiceRegisterRequest,
      params: RequestParams = {},
    ) =>
      this.request<
        ResponseResponse & {
          data?: ServiceUserResponse;
        },
        ResponseResponse
      >({
        path: `/api/auth/register`,
        method: "POST",
        body: request,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 检查系统是否允许注册（公开接口）
     *
     * @tags 设置
     * @name SettingsRegistrationStatusList
     * @summary 获取注册状态
     * @request GET:/api/settings/registration-status
     */
    settingsRegistrationStatusList: (params: RequestParams = {}) =>
      this.request<
        ResponseResponse & {
          data?: Record<string, boolean>;
        },
        any
      >({
        path: `/api/settings/registration-status`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description 获取所有系统设置（需要管理员权限）
     *
     * @tags 设置
     * @name SettingsSystemList
     * @summary 获取系统设置
     * @request GET:/api/settings/system
     * @secure
     */
    settingsSystemList: (params: RequestParams = {}) =>
      this.request<
        ResponseResponse & {
          data?: ServiceSystemSettingsResponse;
        },
        ResponseResponse
      >({
        path: `/api/settings/system`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description 更新系统设置（需要管理员权限）
     *
     * @tags 设置
     * @name SettingsSystemUpdate
     * @summary 更新系统设置
     * @request PUT:/api/settings/system
     * @secure
     */
    settingsSystemUpdate: (
      request: ServiceUpdateSystemSettingsRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseResponse, ResponseResponse>({
        path: `/api/settings/system`,
        method: "PUT",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 修改当前用户的密码
     *
     * @tags 用户
     * @name UserPasswordUpdate
     * @summary 修改密码
     * @request PUT:/api/user/password
     * @secure
     */
    userPasswordUpdate: (
      request: ServiceChangePasswordRequest,
      params: RequestParams = {},
    ) =>
      this.request<ResponseResponse, ResponseResponse>({
        path: `/api/user/password`,
        method: "PUT",
        body: request,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description 获取当前登录用户的详细信息
     *
     * @tags 用户
     * @name UserProfileList
     * @summary 获取用户信息
     * @request GET:/api/user/profile
     * @secure
     */
    userProfileList: (params: RequestParams = {}) =>
      this.request<
        ResponseResponse & {
          data?: ServiceUserResponse;
        },
        ResponseResponse
      >({
        path: `/api/user/profile`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),
  };
}
