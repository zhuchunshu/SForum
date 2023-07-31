<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;
/**
 * @property int $id 
 * @property string $user_id 
 * @property string $old 
 * @property string $new 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UserUsernameChangerLog extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'user_username_changer_log';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'user_id', 'old', 'new', 'created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}