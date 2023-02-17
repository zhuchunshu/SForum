<?php

namespace App\Plugins\User\src\Service\Extend\UserManagement;

use App\Plugins\User\src\Annotation\UserManagementAnnotation;
use App\Plugins\User\src\Service\Extend\UserManagement\Handler\BasisHandler;
use App\Plugins\User\src\Service\Extend\UserManagement\Handler\OptionsHandler;
use App\Plugins\User\src\Service\interfaces\UserManagementInterface;

#[UserManagementAnnotation]
class Options implements UserManagementInterface
{
    public function handler(): string
    {
        return OptionsHandler::class;
    }

    public function edit_view(): string
    {
        return 'User::Admin.Users.management.edit.options';
    }

    public function show_view(): string
    {
        return 'User::Admin.Users.management.show.options';
    }
}