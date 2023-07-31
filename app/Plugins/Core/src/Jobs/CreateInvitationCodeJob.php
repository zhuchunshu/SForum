<?php

namespace App\Plugins\Core\src\Jobs;

use App\Plugins\Core\src\Models\InvitationCode;
use Hyperf\AsyncQueue\Annotation\AsyncQueueMessage;
use Hyperf\Utils\Str;
class CreateInvitationCodeJob
{
    #[AsyncQueueMessage]
    public function handle($count, $after, $before)
    {
        for ($i = 1; $i <= (int) $count; $i++) {
            $code = Str::random(12);
            $code = $before . $code;
            $code .= $after;
            InvitationCode::query()->create(['code' => $code]);
        }
    }
}