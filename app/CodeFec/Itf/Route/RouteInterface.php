<?php


namespace App\CodeFec\Itf\Route;


interface RouteInterface
{
    /**
     * @return array
     */
    public function get(): array;

    /**
     * @param string $route route,path
     * @param $callback
     */
    public function set(string $route, $callback);

}