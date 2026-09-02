use argon2::{
    Argon2,
    password_hash::{PasswordHasher, phc::PasswordHash},
};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let password = rpassword::prompt_password("관리자 비밀번호: ")?;
    let confirmation = rpassword::prompt_password("한 번 더 입력: ")?;
    if password.len() < 12 {
        return Err("비밀번호는 12자 이상이어야 합니다".into());
    }
    if password != confirmation {
        return Err("비밀번호가 일치하지 않습니다".into());
    }
    let hash: PasswordHash = Argon2::default().hash_password(password.as_bytes())?;
    println!("{hash}");
    Ok(())
}
