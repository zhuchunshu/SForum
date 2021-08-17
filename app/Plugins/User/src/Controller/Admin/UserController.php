<?php

namespace App\Plugins\User\src\Controller\Admin;

use App\Plugins\User\src\Models\User;
use Hyperf\HttpServer\Annotation\Controller;

/**
 * Class UserController
 @Controller
 * @package App\Plugins\User\src\Controller
 */
class UserController
{
    public function index(){
        $page = User::query()->paginate(15);
    }
}