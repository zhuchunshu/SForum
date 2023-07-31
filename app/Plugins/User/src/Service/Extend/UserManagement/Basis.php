<?php

namespace App\Plugins\User\src\Service\Extend\UserManagement;

use App\Plugins\User\src\Annotation\UserManagementAnnotation;
use App\Plugins\User\src\Service\Extend\UserManagement\Handler\BasisHandler;
use App\Plugins\User\src\Service\interfaces\UserManagementInterface;
#[UserManagementAnnotation]
class Basis implements UserManagementInterface
{
    public function handler() : string
    {
        return BasisHandler::class;
    }
    public function edit_view() : string
    {
        return 'User::Admin.Users.management.edit.basis';
    }
    public function show_view() : string
    {
        return 'User::Admin.Users.management.show.basis';
    }
}