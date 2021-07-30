<?php

declare(strict_types=1);

namespace App\Plugins\User\src\Mail;

use HyperfExt\Contract\ShouldQueue;
use HyperfExt\Mail\Mailable;

/**
 * 重置密码
 * @package App\Plugins\User\src\Mail
 */
class RePwd extends Mailable implements ShouldQueue
{
    public $data;
    /**
     * Create a new message instance.
     */
    public function __construct($data)
    {
        $this->data = $data;
    }

    /**
     * Build the message.
     */
    public function build(): void
    {
        $this->htmlView('plugins.Core.mail.repwd')->with([
            "data" => $this->data
        ]);
    }
}
