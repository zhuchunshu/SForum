<?php


namespace App\CodeFec\Itf\Theme;


interface ThemeInterface
{
    /**
     * @return array
     */
    public function get(): array;

    /**
     * @param string $namespace route,path
     * @param string $hints
     */
    public function set(string $namespace, string $hints);

}