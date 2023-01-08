<?php

declare(strict_types=1);
/**
 * This file is part of qbhy/hyperf-auth.
 *
 * @link     https://github.com/qbhy/hyperf-auth
 * @document https://github.com/qbhy/hyperf-auth/blob/master/README.md
 * @contact  qbhy0715@qq.com
 * @license  https://github.com/qbhy/hyperf-auth/blob/master/LICENSE
 */
use Qbhy\SimpleJwt\Encoders;
use Qbhy\SimpleJwt\EncryptAdapters as Encrypter;

return [
    'default' => [
        'guard' => 'session',
        'provider' => 'users',
    ],
    'guards' => [
        'sso' => [
            // 支持的设备，env配置时用英文逗号隔开
            'clients' => explode(',', env('AUTH_SSO_CLIENTS', 'pc')),

            // hyperf/redis 实例
            'redis' => function () {
                return make(\Hyperf\Redis\Redis::class);
            },

            // 自定义 redis key，必须包含 {uid}，{uid} 会被替换成用户ID
            'redis_key' => 'u:token:{uid}',

            'driver' => Qbhy\HyperfAuth\Guard\SsoGuard::class,
            'provider' => 'users',

            /*
             * 以下是 simple-jwt 配置
            * 必填
            * jwt 服务端身份标识
            */
            'secret' => env('SSO_JWT_SECRET'),

            /*
             * 可选配置
             * jwt 默认头部token使用的字段
             */
            'header_name' => env('JWT_HEADER_NAME', 'Authorization'),

            /*
             * 可选配置
             * jwt 生命周期，单位秒，默认一天
             */
            'ttl' => (int) env('SIMPLE_JWT_TTL', 60 * 60 * 24),

            /*
             * 可选配置
             * 允许过期多久以内的 token 进行刷新，单位秒，默认一周
             */
            'refresh_ttl' => (int) env('SIMPLE_JWT_REFRESH_TTL', 60 * 60 * 24 * 7),

            /*
             * 可选配置
             * 默认使用的加密类
             */
            'default' => Encrypter\SHA1Encrypter::class,

            /*
             * 可选配置
             * 加密类必须实现 Qbhy\SimpleJwt\Interfaces\Encrypter 接口
             */
            'drivers' => [
                Encrypter\PasswordHashEncrypter::alg() => Encrypter\PasswordHashEncrypter::class,
                Encrypter\CryptEncrypter::alg() => Encrypter\CryptEncrypter::class,
                Encrypter\SHA1Encrypter::alg() => Encrypter\SHA1Encrypter::class,
                Encrypter\Md5Encrypter::alg() => Encrypter\Md5Encrypter::class,
            ],

            /*
             * 可选配置
             * 编码类
             */
            'encoder' => new Encoders\Base64UrlSafeEncoder(),
            //            'encoder' => new Encoders\Base64Encoder(),

            /*
             * 可选配置
             * 缓存类
             */
//            'cache' => new \Doctrine\Common\Cache\FilesystemCache(sys_get_temp_dir()),
            // 如果需要分布式部署，请选择 redis 或者其他支持分布式的缓存驱动
                        'cache' => function () {
                            return make(\Qbhy\HyperfAuth\HyperfRedisCache::class);
                        },

            /*
             * 可选配置
             * 缓存前缀
             */
            'prefix' => env('SIMPLE_JWT_PREFIX', 'default'),
        ],
        'jwt' => [
            'driver' => Qbhy\HyperfAuth\Guard\JwtGuard::class,
            'provider' => 'users',

            /*
             * 以下是 simple-jwt 配置
            * 必填
            * jwt 服务端身份标识
            */
            'secret' => env('SIMPLE_JWT_SECRET'),

            /*
             * 可选配置
             * jwt 默认头部token使用的字段
             */
            'header_name' => env('JWT_HEADER_NAME', 'Authorization'),

            /*
             * 可选配置
             * jwt 生命周期，单位秒，默认一天
             */
            'ttl' => (int) env('SIMPLE_JWT_TTL', 60 * 60 * 24),

            /*
             * 可选配置
             * 允许过期多久以内的 token 进行刷新，单位秒，默认一周
             */
            'refresh_ttl' => (int) env('SIMPLE_JWT_REFRESH_TTL', 60 * 60 * 24 * 7),

            /*
             * 可选配置
             * 默认使用的加密类
             */
            'default' => Encrypter\SHA1Encrypter::class,

            /*
             * 可选配置
             * 加密类必须实现 Qbhy\SimpleJwt\Interfaces\Encrypter 接口
             */
            'drivers' => [
                Encrypter\PasswordHashEncrypter::alg() => Encrypter\PasswordHashEncrypter::class,
                Encrypter\CryptEncrypter::alg() => Encrypter\CryptEncrypter::class,
                Encrypter\SHA1Encrypter::alg() => Encrypter\SHA1Encrypter::class,
                Encrypter\Md5Encrypter::alg() => Encrypter\Md5Encrypter::class,
            ],

            /*
             * 可选配置
             * 编码类
             */
            'encoder' => new Encoders\Base64UrlSafeEncoder(),
            //            'encoder' => new Encoders\Base64Encoder(),

            /*
             * 可选配置
             * 缓存类
             */
            //'cache' => new \Doctrine\Common\Cache\FilesystemCache(sys_get_temp_dir()),
            // 如果需要分布式部署，请选择 redis 或者其他支持分布式的缓存驱动
                        'cache' => function () {
                            return make(\Qbhy\HyperfAuth\HyperfRedisCache::class);
                        },

            /*
             * 可选配置
             * 缓存前缀
             */
            'prefix' => env('SIMPLE_JWT_PREFIX', 'default'),
        ],
        'session' => [
            'driver' => Qbhy\HyperfAuth\Guard\SessionGuard::class,
            'provider' => 'users',
        ],
    ],
    'providers' => [
        'users' => [
            'driver' => \Qbhy\HyperfAuth\Provider\EloquentProvider::class,
            'model' => \App\Plugins\User\src\Models\User::class, //  需要实现 Qbhy\HyperfAuth\Authenticatable 接口
        ],
    ],
];
