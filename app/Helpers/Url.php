<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Helpers;

class Url
{
    /**
     * Generate an absolute URL to the given path.
     *
     * @param \Hyperf\HttpServer\Contract\RequestInterface $request
     * @param string $path
     * @param mixed $extra
     * @return string
     */
    public static function to($request, $path, $extra = [])
    {
        // First we will check if the URL is already a valid URL. If it is we will not
        // try to generate a new one but will simply return the URL as is, which is
        // convenient since developers do not always have to check if it's valid.
        if (self::isValidUrl($path)) {
            return $path;
        }

        $tail = implode(
            '/',
            array_map(
                'rawurlencode',
                (array) self::formatParameters($extra)
            )
        );

        $root = self::getReqFullHost($request);

        [$path, $query] = self::extractQueryString($path);

        return self::format(
            $root,
            '/' . trim($path . '/' . $tail, '/')
        ) . $query;
    }

    /**
     * Determine if the given path is a valid URL.
     *
     * @param string $path
     * @return bool
     */
    public static function isValidUrl($path)
    {
        if (! preg_match('~^(#|//|https?://|mailto:|tel:)~', $path)) {
            return filter_var($path, FILTER_VALIDATE_URL) !== false;
        }
        return true;
    }

    /**
     * Format the array of URL parameters.
     *
     * @param array|mixed $parameters
     * @return array
     */
    public static function formatParameters($parameters)
    {
        return \Hyperf\Utils\Arr::wrap($parameters);
    }

    /**
     * Format the given URL segments into a single URL.
     *
     * @param string $root
     * @param string $path
     * @return string
     */
    public static function format($root, $path)
    {
        $path = '/' . trim($path, '/');
        return trim($root . $path, '/');
    }

    public static function getReqFullHost(\Hyperf\HttpServer\Contract\RequestInterface $serverRequest)
    {
        return self::getReqScheme($serverRequest) . '://' . self::getReqHost($serverRequest);
    }

    public static function getReqScheme(\Hyperf\HttpServer\Contract\RequestInterface $serverRequest)
    {
        $headers = $serverRequest->getHeaders();
        if (isset($headers['x-scheme'])) {
            return $headers['x-scheme'][0];
        }
        if (isset($headers['x-forwarded-proto'])) {
            return $headers['x-forwarded-proto'][0];
        }
        return ($serverRequest->getUri()->getScheme() === 'https') ? 'https' : 'http';
    }

    public static function getReqHost(\Hyperf\HttpServer\Contract\RequestInterface $serverRequest)
    {
        //$host = strtr($host, [':9501' => '']);
        return $serverRequest->header('host');
    }

    /**
     * Extract the query string from the given path.
     *
     * @param string $path
     * @return array
     */
    protected static function extractQueryString($path)
    {
        if (($queryPosition = strpos($path, '?')) !== false) {
            return [
                substr($path, 0, $queryPosition),
                substr($path, $queryPosition),
            ];
        }

        return [$path, ''];
    }
}

function url(): Url
{
    return new \App\Helpers\Url();
}
